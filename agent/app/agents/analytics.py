import json
import logging
from typing import Any, Literal, Protocol

from pydantic import ValidationError

from app.broker.backend_rpc import BackendRpcError
from app.schemas.analytics import (
    AnalyticsContext,
    ConstructionDelayContext,
    ConstructionEventsData,
)
from app.schemas.client_facts import ClientFacts, DealMessagesData
from app.schemas.dialog import DialogAnalyzeResult
from app.schemas.negotiation import (
    ApartmentFacts,
    BuildingFacts,
    DealFacts,
    require_backend_data,
)
from app.tools.apartment import ApartmentTools
from app.tools.building import BuildingTools
from app.tools.client import ClientTools
from app.tools.construction import ConstructionTools
from app.tools.deal import DealTools
from app.tools.messages import MessagesTools

logger = logging.getLogger(__name__)

AnalyticsIntent = Literal[
    "analyze_risk",
    "analyze_construction",
    "extract_client_facts",
]

MISSING_DEAL_RESPONSE = (
    "Для анализа строительных рисков необходимо выбрать или передать сделку. "
    "Без данных по сделке и объекту нельзя подготовить фактический анализ."
)
BACKEND_DATA_ERROR_RESPONSE = (
    "Не удалось получить актуальные данные по объекту. "
    "Для анализа рисков нужны фактические данные строительства."
)
CLIENT_FACTS_MISSING_DEAL_RESPONSE = (
    "Для анализа переписки необходимо выбрать или передать сделку."
)
EMPTY_MESSAGES_RESPONSE = (
    "В истории сделки пока недостаточно сообщений для анализа предпочтений клиента."
)
CLIENT_FACTS_ERROR_RESPONSE = (
    "Не удалось получить или корректно проанализировать историю переписки клиента."
)
CLIENT_FACTS_SAVE_ERROR_RESPONSE = (
    "Карточку клиента обновить не удалось."
)
CLIENT_FACTS_SAVE_SUCCESS_RESPONSE = "Извлечённые данные сохранены в карточке клиента."

ANALYTICS_SYSTEM_PROMPT = """Ты аналитический помощник менеджера по продажам недвижимости.
Подготовь краткий практический анализ строительных рисков по конкретной сделке.

Строгие правила:
- Используй только данные из секции «Фактический контекст backend» и текст менеджера.
- ЗАПРЕЩЕНО придумывать задержки, даты, причины событий и risk level.
- completion_percentage — фактическая готовность этапа. Упоминай её только если
  поле присутствует; если оно null, не придумывай процент готовности.
- Называй точную дату сдачи только когда exact_delivery_date_allowed=true — это
  «зелёная зона», без зафиксированных средних/высоких рисков и задержек.
- Если exact_delivery_date_allowed=false, не обещай и не называй точную дату:
  сообщи, что срок требует уточнения из-за зарегистрированного риска.
- Если building.planned_delivery=null, не придумывай плановую дату сдачи.
- ЗАПРЕЩЕНО утверждать влияние на срок сдачи без фактических оснований в context.
- Не выполняй финансовые расчёты и не добавляй сведения из внешних источников.
- Если данных недостаточно, явно скажи об этом менеджеру.

Intent описывает задачу:
- analyze_risk — выявить и объяснить риски для сделки;
- analyze_construction — оценить текущее состояние строительства по фактам.

Структура ответа:
1. Текущее состояние.
2. Выявленные риски.
3. Возможное влияние на сделку.
4. Рекомендация менеджеру.
"""

CLIENT_FACTS_PROMPT = """Извлеки факты о клиенте из его сообщений и верни structured output.

Строгие правила:
- Источником фактов являются только сообщения с direction=client_to_manager.
- Не считай вопросы, варианты и предположения менеджера предпочтениями клиента.
- Заполняй поле только если факт явно указан или надёжно следует из слов клиента.
- Не додумывай бюджет, комнаты, этаж, парковку, ремонт, район или сроки покупки.
- Верни ВСЕ поля схемы, не ограничивайся одним полем summary.
- Для неизвестных одиночных значений явно используй null, для списков — [].
- summary должна кратко описывать только факты, присутствующие в сообщениях клиента.

Сообщения клиента:
{messages}
"""

CLIENT_FACTS_JSON_FALLBACK_PROMPT = """Извлеки подтверждённые факты о клиенте
только из сообщений с direction=client_to_manager ниже.

Верни ТОЛЬКО один JSON object:
- без markdown;
- без ```json и других code fences;
- без комментариев;
- без текста до или после JSON.

JSON object обязан содержать ВСЕ поля:
- budget_min
- budget_max
- rooms
- floor_min
- floor_max
- parking_required
- renovation_required
- preferred_district
- purchase_timeline
- important_factors
- objections
- summary

Строгие правила:
- Не считай вопросы, варианты и предположения менеджера предпочтениями клиента.
- Заполняй поле только если факт явно указан или надёжно следует из слов клиента.
- Не додумывай бюджет, комнаты, этаж, парковку, ремонт, район или сроки покупки.
- Для неизвестного nullable scalar используй null.
- Для неизвестного list используй [].
- summary должна описывать только факты из сообщений клиента.

Сообщения клиента:
{messages}
"""

CONSTRUCTION_DELAY_SYSTEM_PROMPT = """Ты аналитический помощник менеджера по недвижимости.
Сформируй короткую практическую рекомендацию по активной сделке на основе события
о задержке строительства.

Строгие правила:
- Используй только переданный factual context.
- ЗАПРЕЩЕНО придумывать сроки, причины задержки и affected clients.
- ЗАПРЕЩЕНО изменять или придумывать risk level и delay_days.
- Не добавляй факты, которых нет в context.
- Объясни риск, возможное влияние на сделку и рекомендуемое действие менеджера.
"""


class AnalyticsLlmClient(Protocol):
    async def generate(
        self,
        message: str,
        *,
        system_prompt: str | None = None,
    ) -> str: ...

    async def parse_structured(
        self,
        prompt: str,
        response_model: type[ClientFacts],
    ) -> ClientFacts: ...

class AnalyticsAgent:
    def __init__(
        self,
        llm: AnalyticsLlmClient,
        deal_tools: DealTools,
        apartment_tools: ApartmentTools,
        building_tools: BuildingTools,
        construction_tools: ConstructionTools,
        messages_tools: MessagesTools,
        client_tools: ClientTools,
    ) -> None:
        self._llm = llm
        self._deal_tools = deal_tools
        self._apartment_tools = apartment_tools
        self._building_tools = building_tools
        self._construction_tools = construction_tools
        self._messages_tools = messages_tools
        self._client_tools = client_tools

    async def generate(
        self,
        message: str,
        intent: AnalyticsIntent,
        deal_id: int | None,
    ) -> str:
        if intent not in (
            "analyze_risk",
            "analyze_construction",
            "extract_client_facts",
        ):
            raise ValueError(f"Unsupported analytics intent: {intent}")
        if intent == "extract_client_facts":
            return await self._extract_client_facts(deal_id, message)
        if deal_id is None:
            return await self._llm.generate(
                message,
                system_prompt=ANALYTICS_SYSTEM_PROMPT,
            )

        try:
            deal_response = await self._deal_tools.get_deal(deal_id)
            deal = DealFacts.model_validate(require_backend_data(deal_response.data))

            apartment_response = await self._apartment_tools.get_apartment(
                deal.apartment_id
            )
            apartment = ApartmentFacts.model_validate(
                require_backend_data(apartment_response.data)
            )

            building_response = await self._building_tools.get_building(
                apartment.building_id
            )
            building = BuildingFacts.model_validate(
                require_backend_data(building_response.data)
            )

            events_response = await self._construction_tools.get_construction_events(
                apartment.building_id
            )
            events = ConstructionEventsData.model_validate(
                require_backend_data(events_response.data)
            )
        except BackendRpcError as exc:
            logger.warning(
                "Analytics backend data unavailable: error_type=%s code=%s",
                type(exc).__name__,
                exc.code,
            )
            return BACKEND_DATA_ERROR_RESPONSE
        except (ValidationError, TypeError, ValueError) as exc:
            logger.warning(
                "Analytics backend data is invalid: error_type=%s",
                type(exc).__name__,
            )
            return BACKEND_DATA_ERROR_RESPONSE

        exact_delivery_date_allowed = building.planned_delivery is not None and all(
            event.risk_level == "low" and not event.delay_days
            for event in events.events
        )
        safe_building = building if exact_delivery_date_allowed else building.model_copy(
            update={"planned_delivery": None, "forecast_delivery": None}
        )
        context = AnalyticsContext(
            deal=deal,
            apartment=apartment,
            building=safe_building,
            construction_events=events.events,
            exact_delivery_date_allowed=exact_delivery_date_allowed,
        )

        user_prompt = (
            f"Intent: {intent}\n\n"
            "Фактический контекст backend:\n"
            f"{context.model_dump_json(indent=2)}\n\n"
            "Текст менеджера:\n"
            f"{message}"
        )
        return await self._llm.generate(
            user_prompt,
            system_prompt=ANALYTICS_SYSTEM_PROMPT,
        )

    async def generate_delay_recommendation(
        self,
        deal: DealFacts,
        *,
        building_id: int,
        delay_days: int,
        risk_level: Literal["low", "medium", "high"],
    ) -> str:
        context = ConstructionDelayContext(
            deal=deal,
            building_id=building_id,
            delay_days=delay_days,
            risk_level=risk_level,
        )
        prompt = (
            "Фактический контекст:\n"
            f"{context.model_dump_json(indent=2)}"
        )
        return await self._llm.generate(
            prompt,
            system_prompt=CONSTRUCTION_DELAY_SYSTEM_PROMPT,
        )

    async def _extract_client_facts(
        self,
        deal_id: int | None,
        message: str = "",
    ) -> str:
        if deal_id is None:
            if message:
                try:
                    facts = await self._extract_structured_client_facts(message)
                    return self._format_client_facts(facts)
                except Exception as exc:
                    logger.warning("Direct client facts extraction fallback: %s", exc)
                    return await self._llm.generate(
                        f"Проанализируй диалог и выдели ключевые потребности, бюджет и сомнения клиента:\n\n{message}",
                        system_prompt="Ты аналитический помощник менеджера по продажам недвижимости ДСК. Проанализируй переписку и выдели подтверждённые потребности, бюджет, сомнения и ключевые критерии клиента.",
                    )
            return EMPTY_MESSAGES_RESPONSE

        try:
            result = await self.analyze_dialog(deal_id)
        except NoClientMessagesError:
            return EMPTY_MESSAGES_RESPONSE
        except BackendRpcError as exc:
            logger.warning(
                "Client messages unavailable: error_type=%s code=%s",
                type(exc).__name__,
                exc.code,
            )
            return CLIENT_FACTS_ERROR_RESPONSE
        except (ValidationError, TypeError, ValueError) as exc:
            logger.warning(
                "Client messages response is invalid: error_type=%s",
                type(exc).__name__,
            )
            return CLIENT_FACTS_ERROR_RESPONSE

        facts = result.analysis
        summary = self._format_client_facts(facts)
        should_update = facts.budget_max is not None or bool(
            self._build_preferences(facts)
        )
        if should_update:
            suffix = (
                CLIENT_FACTS_SAVE_SUCCESS_RESPONSE
                if result.preferences_updated
                else CLIENT_FACTS_SAVE_ERROR_RESPONSE
            )
            return f"{summary}\n\n{suffix}"
        return summary

    async def analyze_dialog(self, deal_id: int) -> DialogAnalyzeResult:
        deal_response = await self._deal_tools.get_deal(deal_id)
        deal = DealFacts.model_validate(require_backend_data(deal_response.data))
        if deal.client_id is None:
            raise ValueError("deal does not contain client_id")

        messages_response = await self._messages_tools.get_deal_messages(deal_id)
        messages_data = DealMessagesData.model_validate(
            require_backend_data(messages_response.data)
        )
        client_messages = [
            message
            for message in messages_data.messages
            if message.direction == "client_to_manager"
        ]
        if not client_messages:
            raise NoClientMessagesError("dialog does not contain client messages")

        messages_json = json.dumps(
            [
                {
                    "direction": message.direction,
                    "body": message.body,
                }
                for message in client_messages
            ],
            ensure_ascii=False,
            indent=2,
        )
        facts = await self._extract_structured_client_facts(messages_json)

        preferences = self._build_preferences(facts)
        preferences_updated = False
        if facts.budget_max is not None or preferences:
            try:
                await self._client_tools.update_client_preferences(
                    deal.client_id,
                    preferences=preferences,
                    budget_max=facts.budget_max,
                )
            except BackendRpcError as exc:
                logger.warning(
                    "Client preferences update failed: error_type=%s code=%s",
                    type(exc).__name__,
                    exc.code,
                )
            else:
                preferences_updated = True

        return DialogAnalyzeResult(
            deal_id=deal_id,
            client_id=deal.client_id,
            analysis=facts,
            preferences_updated=preferences_updated,
        )

    async def _extract_structured_client_facts(
        self,
        messages_json: str,
    ) -> ClientFacts:
        primary_prompt = CLIENT_FACTS_PROMPT.format(messages=messages_json)
        try:
            parsed = await self._llm.parse_structured(primary_prompt, ClientFacts)
            return ClientFacts.model_validate(parsed)
        except (json.JSONDecodeError, ValidationError, ValueError) as exc:
            logger.warning(
                "Native structured client facts failed, using JSON fallback: "
                "error_type=%s",
                type(exc).__name__,
            )

        fallback_prompt = CLIENT_FACTS_JSON_FALLBACK_PROMPT.format(
            messages=messages_json
        )
        try:
            raw_response = await self._llm.generate(fallback_prompt)
            decoded = json.loads(raw_response.strip())
            return ClientFacts.model_validate(decoded)
        except (json.JSONDecodeError, ValidationError, TypeError, ValueError) as exc:
            logger.error(
                "Client facts JSON fallback failed: error_type=%s",
                type(exc).__name__,
            )
            raise ClientFactsExtractionError(
                "GigaChat returned invalid client facts in both extraction paths"
            ) from exc

    @staticmethod
    def _build_preferences(facts: ClientFacts) -> dict[str, Any]:
        raw_preferences = facts.model_dump(
            exclude={"budget_min", "budget_max", "summary"},
            exclude_none=True,
        )
        preferences = {
            key: value
            for key, value in raw_preferences.items()
            if not isinstance(value, list) or value
        }
        if "parking_required" in preferences:
            preferences["parking"] = preferences.pop("parking_required")
        return preferences

    @staticmethod
    def _format_client_facts(facts: ClientFacts) -> str:
        details: list[str] = []
        if facts.budget_min is not None and facts.budget_max is not None:
            details.append(
                "бюджет: "
                f"{AnalyticsAgent._format_money(facts.budget_min)}–"
                f"{AnalyticsAgent._format_money(facts.budget_max)} ₽"
            )
        elif facts.budget_max is not None:
            details.append(
                f"бюджет: до {AnalyticsAgent._format_money(facts.budget_max)} ₽"
            )
        elif facts.budget_min is not None:
            details.append(
                f"бюджет: от {AnalyticsAgent._format_money(facts.budget_min)} ₽"
            )
        if facts.rooms is not None:
            details.append(f"комнат: {facts.rooms}")
        if facts.floor_min is not None and facts.floor_max is not None:
            details.append(f"этаж: {facts.floor_min}–{facts.floor_max}")
        elif facts.floor_min is not None:
            details.append(f"этаж: от {facts.floor_min}")
        elif facts.floor_max is not None:
            details.append(f"этаж: до {facts.floor_max}")
        if facts.parking_required is not None:
            value = "обязательна" if facts.parking_required else "не обязательна"
            details.append(f"парковка: {value}")
        if facts.renovation_required is not None:
            value = "нужна" if facts.renovation_required else "не нужна"
            details.append(f"отделка: {value}")
        if facts.preferred_district is not None:
            details.append(f"предпочтительный район: {facts.preferred_district}")
        if facts.purchase_timeline is not None:
            details.append(f"срок покупки: {facts.purchase_timeline}")
        if facts.important_factors:
            details.append(f"важные факторы: {', '.join(facts.important_factors)}")

        sections = [f"Краткая сводка: {facts.summary}"]
        if details:
            sections.append(
                "По переписке удалось определить:\n"
                + "\n".join(f"- {detail}" for detail in details)
            )
        if facts.objections:
            sections.append(
                "Возражения клиента:\n"
                + "\n".join(f"- {objection}" for objection in facts.objections)
            )
        return "\n\n".join(sections)

    @staticmethod
    def _format_money(value: int) -> str:
        return f"{value:,}".replace(",", " ")

class NoClientMessagesError(ValueError):
    pass

class ClientFactsExtractionError(ValueError):
    pass