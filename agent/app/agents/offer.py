import asyncio
import json
import logging
import re
import time
from dataclasses import dataclass
from decimal import Decimal
from typing import Literal

from pydantic import ValidationError

from app.graphs.offer_graph import (
    BACKEND_DATA_ERROR_RESPONSE,
    OFFER_SAVE_ERROR_RESPONSE,
    OFFER_SYSTEM_PROMPT,
    OfferGraph,
    OfferLlmClient,
)
from app.graphs.offer_state import OfferState
from app.schemas.offer import OfferRequestFacts
from app.tools.apartment import ApartmentTools
from app.tools.client import ClientTools
from app.tools.deal import DealTools
from app.tools.offer import OfferTools

OfferIntent = Literal["create_offer", "calculate_offer"]

DEFAULT_DISCOUNT_PERCENT = 0
MISSING_DEAL_RESPONSE = (
    "Для формирования предложения необходимо выбрать или передать сделку."
)
DISCOUNT_CLARIFICATION_RESPONSE = "Какой процент скидки вы хотите указать?"
DISCOUNT_RANGE_RESPONSE = "Укажите процент скидки от 0 до 100."
DISCOUNT_PRECISION_RESPONSE = (
    "Укажите процент скидки максимум с двумя знаками после запятой."
)
DISCOUNT_EXTRACTION_RESPONSE = (
    "Не удалось определить процент скидки. Укажите его числом, например 7%."
)
PENDING_OFFER_TTL_SECONDS = 15 * 60
MAX_PENDING_OFFERS = 1000

OFFER_REQUEST_EXTRACTION_PROMPT = """Извлеки из запроса менеджера только явно
указанную скидку для коммерческого предложения.

Правила:
- discount_mentioned=true, если менеджер просит скидку, включая просьбы без числа.
- requested_discount_percent содержит только явно написанный процент скидки.
- Не придумывай, не подбирай и не оптимизируй процент.
- Не принимай цену, бюджет, ставку ипотеки или backend-лимит за скидку.
- Если скидка не упомянута: discount_mentioned=false и requested_discount_percent=null.
- Если скидка упомянута без конкретного числа: discount_mentioned=true и requested_discount_percent=null.
- Русскую десятичную запятую нормализуй: 5,5% означает 5.5.
- Верни structured result по переданной схеме.
{follow_up_rule}

Запрос менеджера:
{message}
"""

logger = logging.getLogger(__name__)

_PERCENT_PATTERN = re.compile(
    r"(?<![\w.,])([+-]?\d+(?:[.,]\d+)?)\s*(?:%|процент(?:а|ов)?)(?!\w)",
    re.IGNORECASE,
)


@dataclass(frozen=True)
class _PendingOffer:
    intent: OfferIntent
    deal_id: int
    original_message: str
    parking_unit_id: int | None
    storage_unit_id: int | None
    created_at: float


@dataclass(frozen=True)
class _DiscountDecision:
    value: int | float | None = None
    response: str | None = None


class OfferAgent:
    def __init__(
        self,
        llm: OfferLlmClient,
        deal_tools: DealTools,
        client_tools: ClientTools,
        apartment_tools: ApartmentTools,
        offer_tools: OfferTools,
    ) -> None:
        self._llm = llm
        self._graph = OfferGraph(
            llm,
            deal_tools,
            client_tools,
            apartment_tools,
            offer_tools,
        )
        self._pending_offers: dict[tuple[int, int], _PendingOffer] = {}
        self._pending_lock = asyncio.Lock()

    async def generate(
        self,
        message: str,
        intent: OfferIntent,
        deal_id: int | None,
        user_id: int,
        session_id: int | None = None,
        parking_unit_id: int | None = None,
        storage_unit_id: int | None = None,
    ) -> str:
        if intent not in ("create_offer", "calculate_offer"):
            raise ValueError(f"Unsupported offer intent: {intent}")
        if deal_id is None:
            return await self._llm.generate(
                message,
                system_prompt=OFFER_SYSTEM_PROMPT,
            )

        discount = await self._extract_discount(message, follow_up=False)
        if discount.response is not None:
            await self._remember_pending(
                user_id,
                session_id,
                intent,
                deal_id,
                message,
                parking_unit_id,
                storage_unit_id,
            )
            return discount.response

        await self._clear_pending(user_id, session_id)
        return await self._run_graph(
            message,
            intent,
            deal_id,
            user_id,
            discount.value,
            parking_unit_id,
            storage_unit_id,
        )

    async def resume_pending(
        self,
        message: str,
        user_id: int,
        session_id: int,
    ) -> tuple[str, OfferIntent] | None:
        pending = await self._get_pending(user_id, session_id)
        if pending is None:
            return None

        discount = await self._extract_discount(message, follow_up=True)
        if discount.response is not None:
            return discount.response, pending.intent

        await self._clear_pending(user_id, session_id)
        result = await self._run_graph(
            f"{pending.original_message}\nУточнённый процент скидки: {message}",
            pending.intent,
            pending.deal_id,
            user_id,
            discount.value,
            pending.parking_unit_id,
            pending.storage_unit_id,
        )
        return result, pending.intent

    async def _run_graph(
        self,
        message: str,
        intent: OfferIntent,
        deal_id: int,
        user_id: int,
        requested_discount_percent: int | float | None,
        parking_unit_id: int | None,
        storage_unit_id: int | None,
    ) -> str:
        if requested_discount_percent is None:
            raise RuntimeError("Offer discount decision is incomplete")
        initial_state: OfferState = {
            "user_id": user_id,
            "deal_id": deal_id,
            "message": message,
            "intent": intent,
            "requested_discount_percent": requested_discount_percent,
            "parking_unit_id": parking_unit_id,
            "storage_unit_id": storage_unit_id,
            "approval_required": False,
        }
        result = await self._graph.run(initial_state)
        return result["result_message"]

    async def _extract_discount(
        self,
        message: str,
        *,
        follow_up: bool,
    ) -> _DiscountDecision:
        discount_context = follow_up or "скидк" in message.casefold()
        explicit_values = self._explicit_percentages(message)
        if discount_context:
            invalid_response = self._validate_explicit_values(explicit_values)
            if invalid_response is not None:
                return _DiscountDecision(response=invalid_response)

        prompt = OFFER_REQUEST_EXTRACTION_PROMPT.format(
            follow_up_rule=(
                "Это ответ на уточнение процента скидки; самостоятельное значение "
                "вида 7% считай явно указанной скидкой."
                if follow_up
                else ""
            ),
            message=message,
        )
        facts: OfferRequestFacts | None = None
        for attempt in range(1, 3):
            try:
                facts = await self._llm.parse_structured(prompt, OfferRequestFacts)
                break
            except (json.JSONDecodeError, ValidationError, ValueError) as exc:
                logger.warning(
                    "Invalid structured offer request: attempt=%d/2 error_type=%s",
                    attempt,
                    type(exc).__name__,
                )

        if facts is None:
            return _DiscountDecision(response=DISCOUNT_EXTRACTION_RESPONSE)

        if not facts.discount_mentioned:
            if discount_context:
                return _DiscountDecision(response=DISCOUNT_CLARIFICATION_RESPONSE)
            return _DiscountDecision(value=DEFAULT_DISCOUNT_PERCENT)

        requested = facts.requested_discount_percent
        if requested is None:
            return _DiscountDecision(response=DISCOUNT_CLARIFICATION_RESPONSE)

        invalid_response = self._validate_explicit_values(explicit_values)
        if invalid_response is not None:
            return _DiscountDecision(response=invalid_response)

        unique_values = set(explicit_values)
        if len(unique_values) != 1 or requested not in unique_values:
            return _DiscountDecision(response=DISCOUNT_CLARIFICATION_RESPONSE)

        return _DiscountDecision(value=self._json_number(requested))

    @staticmethod
    def _explicit_percentages(message: str) -> list[Decimal]:
        return [
            Decimal(match.group(1).replace(",", "."))
            for match in _PERCENT_PATTERN.finditer(message)
        ]

    @staticmethod
    def _validate_explicit_values(values: list[Decimal]) -> str | None:
        for value in values:
            if value < 0 or value > 100:
                return DISCOUNT_RANGE_RESPONSE
            decimal_places = max(0, -value.as_tuple().exponent)
            if decimal_places > 2:
                return DISCOUNT_PRECISION_RESPONSE
        return None

    @staticmethod
    def _json_number(value: Decimal) -> int | float:
        return int(value) if value == value.to_integral_value() else float(value)

    async def _remember_pending(
        self,
        user_id: int,
        session_id: int | None,
        intent: OfferIntent,
        deal_id: int,
        message: str,
        parking_unit_id: int | None,
        storage_unit_id: int | None,
    ) -> None:
        if session_id is None:
            return
        async with self._pending_lock:
            self._purge_expired_locked()
            if len(self._pending_offers) >= MAX_PENDING_OFFERS:
                oldest = min(
                    self._pending_offers,
                    key=lambda key: self._pending_offers[key].created_at,
                )
                self._pending_offers.pop(oldest, None)
            self._pending_offers[(user_id, session_id)] = _PendingOffer(
                intent=intent,
                deal_id=deal_id,
                original_message=message,
                parking_unit_id=parking_unit_id,
                storage_unit_id=storage_unit_id,
                created_at=time.monotonic(),
            )

    async def _get_pending(
        self,
        user_id: int,
        session_id: int,
    ) -> _PendingOffer | None:
        async with self._pending_lock:
            self._purge_expired_locked()
            return self._pending_offers.get((user_id, session_id))

    async def _clear_pending(self, user_id: int, session_id: int | None) -> None:
        if session_id is None:
            return
        async with self._pending_lock:
            self._pending_offers.pop((user_id, session_id), None)

    def _purge_expired_locked(self) -> None:
        threshold = time.monotonic() - PENDING_OFFER_TTL_SECONDS
        expired = [
            key
            for key, pending in self._pending_offers.items()
            if pending.created_at < threshold
        ]
        for key in expired:
            self._pending_offers.pop(key, None)
