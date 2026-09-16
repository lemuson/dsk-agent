import json
import logging
from typing import Literal, Protocol

from pydantic import ValidationError

from app.broker.backend_rpc import BackendRpcError
from app.schemas.negotiation import (
    ApartmentFacts,
    BuildingFacts,
    CompetitorListData,
    DealFacts,
    NegotiationContext,
    require_backend_data,
)
from app.schemas.dialog import ReplyAssistAnalysis
from app.tools.apartment import ApartmentTools
from app.tools.building import BuildingTools
from app.tools.competitor import CompetitorTools
from app.tools.deal import DealTools

logger = logging.getLogger(__name__)

NegotiationIntent = Literal["handle_objection", "compare_competitor"]

MISSING_DEAL_RESPONSE = (
    "Для точного анализа необходимо выбрать или передать сделку. "
    "Без данных по сделке и объекту нельзя формировать фактический контраргумент."
)
BACKEND_DATA_ERROR_RESPONSE = (
    "Не удалось получить данные по сделке или объекту. "
    "Для точного контраргумента нужны фактические данные."
)
GROUNDING_FALLBACK_RESPONSE = (
    "Недостаточно подтверждённых данных для безопасного контраргумента. "
    "Уточните фактические характеристики нашего объекта и конкурента."
)

_FACTUAL_TOPIC_ROOTS = (
    "планиров",
    "инфраструктур",
    "транспорт",
    "ипотек",
    "скидк",
    "рассрочк",
    "срок",
    "сдач",
    "цен",
    "площад",
    "отделк",
    "парков",
    "метро",
    "школ",
    "магазин",
    "эколог",
    "благоустрой",
)

NEGOTIATION_SYSTEM_PROMPT = """Ты помощник менеджера по продажам недвижимости.
Сформируй короткий практический ответ для менеджера по продажам.

Строгие правила:
- ФАКТЫ — только данные из секции «Фактический контекст backend» и текст менеджера.
- Текст менеджера может содержать слова клиента; не выдавай их за проверенные данные.
- ЗАПРЕЩЕНО придумывать цены, скидки и любые финансовые показатели.
- ЗАПРЕЩЕНО придумывать сроки строительства или сдачи.
- Этот сценарий не получает подтверждение «зелёной зоны», поэтому точную дату
  сдачи называть запрещено даже при вопросе клиента о сроках.
- ЗАПРЕЩЕНО придумывать инфраструктуру, преимущества нашего объекта или недостатки
  конкурента.
- ЗАПРЕЩЕНО ссылаться на сведения, которых нет в фактическом контексте.
- Используй только факты из factual context. Если преимущество не подтверждено
  данными, не упоминай его.
- Не расширяй и не подменяй смысл факта: «варианты отделки» нельзя превращать
  в «разнообразие планировок» или объединять с планировками.
- Инфраструктуру, транспорт, ипотеку, скидки, сроки, площадь, отделку, парковку
  и любые другие преимущества можно упоминать только при их явном наличии в
  factual context. Сохраняй принадлежность факта нашему объекту или конкуренту.
- Если данных недостаточно, явно скажи об этом менеджеру.

Intent описывает задачу:
- handle_objection — помочь корректно обработать возражение клиента;
- compare_competitor — помочь сравнить предложение с конкурентом без домыслов.

Желательная структура:
1. Краткая оценка возражения.
2. От двух до четырёх фактических контраргументов, если данных достаточно.
3. Готовая формулировка, которую менеджер может сказать клиенту.

Не выполняй расчёты и не сообщай неподтверждённые сведения.
"""

REPLY_ASSIST_ANALYSIS_PROMPT = """Определи тип и кратко опиши конкретное сообщение клиента.
Верни только structured output по переданной схеме.

Допустимые intent:
- price_objection — клиент возражает против цены;
- competitor_comparison — клиент сравнивает предложение с конкурентом;
- timing_objection — вопрос или возражение о сроках;
- property_objection — вопрос или возражение о характеристиках объекта;
- general_question — общий вопрос клиента;
- other_objection — другое возражение;
- unknown — смысл нельзя надёжно определить.

Строгие правила:
- Анализируй только текст сообщения ниже.
- Не придумывай факты о клиенте, цене, скидке, сроках, объекте или конкуренте.
- summary должна кратко пересказывать смысл сообщения, без новых сведений.

Сообщение клиента:
{message}
"""


class NegotiationLlmClient(Protocol):
    async def generate(
        self,
        message: str,
        *,
        system_prompt: str | None = None,
    ) -> str: ...

    async def parse_structured(
        self,
        prompt: str,
        response_model: type[ReplyAssistAnalysis],
    ) -> ReplyAssistAnalysis: ...


class NegotiationAgent:
    def __init__(
        self,
        llm: NegotiationLlmClient,
        deal_tools: DealTools,
        apartment_tools: ApartmentTools,
        building_tools: BuildingTools,
        competitor_tools: CompetitorTools,
    ) -> None:
        self._llm = llm
        self._deal_tools = deal_tools
        self._apartment_tools = apartment_tools
        self._building_tools = building_tools
        self._competitor_tools = competitor_tools

    async def generate(
        self,
        message: str,
        intent: NegotiationIntent,
        deal_id: int | None,
    ) -> str:
        if intent not in ("handle_objection", "compare_competitor"):
            raise ValueError(f"Unsupported negotiation intent: {intent}")
        if deal_id is None:
            return await self._llm.generate(
                message,
                system_prompt=NEGOTIATION_SYSTEM_PROMPT,
            )

        try:
            deal_response = await self._deal_tools.get_deal(deal_id)
            deal = DealFacts.model_validate(require_backend_data(deal_response.data))
            context = await self._load_context(deal)
        except BackendRpcError as exc:
            logger.warning(
                "Negotiation backend data unavailable: error_type=%s code=%s",
                type(exc).__name__,
                exc.code,
            )
            return BACKEND_DATA_ERROR_RESPONSE
        except (ValidationError, TypeError, ValueError) as exc:
            logger.warning(
                "Negotiation backend data is invalid: error_type=%s",
                type(exc).__name__,
            )
            return BACKEND_DATA_ERROR_RESPONSE

        return await self._generate_from_context(message, intent, context)

    async def reply_assist(
        self,
        message: str,
        deal: DealFacts,
    ) -> tuple[ReplyAssistAnalysis, str]:
        analysis: ReplyAssistAnalysis | None = None
        prompt = REPLY_ASSIST_ANALYSIS_PROMPT.format(message=message)
        for attempt in range(1, 3):
            try:
                parsed = await self._llm.parse_structured(prompt, ReplyAssistAnalysis)
                analysis = ReplyAssistAnalysis.model_validate(parsed)
                break
            except (json.JSONDecodeError, ValidationError, ValueError) as exc:
                logger.warning(
                    "Invalid structured reply analysis: attempt=%d/2 error_type=%s",
                    attempt,
                    type(exc).__name__,
                )
        if analysis is None:
            analysis = ReplyAssistAnalysis(
                intent="unknown",
                summary="Смысл сообщения не удалось надёжно классифицировать.",
            )

        intent: NegotiationIntent = (
            "compare_competitor"
            if analysis.intent == "competitor_comparison"
            else "handle_objection"
        )
        context = await self._load_context(deal)
        suggested_reply = await self._generate_from_context(message, intent, context)
        return analysis, suggested_reply

    async def _load_context(self, deal: DealFacts) -> NegotiationContext:
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
        building = building.model_copy(
            update={"planned_delivery": None, "forecast_delivery": None}
        )

        competitors_response = await self._competitor_tools.list_competitors(
            building.district
        )
        competitors = CompetitorListData.model_validate(
            require_backend_data(competitors_response.data)
        )
        return NegotiationContext(
            deal=deal,
            apartment=apartment,
            building=building,
            competitors=competitors.competitors,
        )

    async def _generate_from_context(
        self,
        message: str,
        intent: NegotiationIntent,
        context: NegotiationContext,
    ) -> str:
        allowed_facts = self._format_allowed_facts(context)
        user_prompt = (
            f"Intent: {intent}\n\n"
            "Фактический контекст backend:\n"
            f"{context.model_dump_json(indent=2)}\n\n"
            "Разрешённые факты (не расширять и не подменять формулировки):\n"
            f"{allowed_facts}\n\n"
            "Текст менеджера:\n"
            f"{message}"
        )
        response = await self._llm.generate(
            user_prompt,
            system_prompt=NEGOTIATION_SYSTEM_PROMPT,
        )
        unsupported = self._unsupported_topics(response, user_prompt)
        if not unsupported:
            return response

        logger.warning(
            "Negotiation response contains unsupported factual topics; retrying: "
            "topics=%s",
            ",".join(unsupported),
        )
        retry_prompt = (
            f"{user_prompt}\n\n"
            "Предыдущий ответ содержал неподтверждённые темы. Сформируй ответ заново "
            "и используй только перечисленные разрешённые факты."
        )
        response = await self._llm.generate(
            retry_prompt,
            system_prompt=NEGOTIATION_SYSTEM_PROMPT,
        )
        if self._unsupported_topics(response, user_prompt):
            logger.error("Negotiation grounding validation failed after retry")
            return GROUNDING_FALLBACK_RESPONSE
        return response

    @staticmethod
    def _format_allowed_facts(context: NegotiationContext) -> str:
        facts = [
            f"Квартира: id={context.apartment.id}.",
            f"Здание: id={context.building.id}.",
            f"Район здания: {context.building.district}.",
        ]
        apartment_fields = (
            ("Номер квартиры", context.apartment.number),
            ("Этаж квартиры", context.apartment.floor),
            ("Количество комнат", context.apartment.rooms),
            ("Площадь квартиры", context.apartment.area),
            ("Цена квартиры", context.apartment.price),
            ("Статус квартиры", context.apartment.status),
        )
        building_fields = (
            ("Название здания", context.building.name),
            ("Готовность здания", context.building.readiness_percent),
            ("Плановая дата сдачи", context.building.planned_delivery),
            ("Прогнозная дата сдачи", context.building.forecast_delivery),
        )
        for label, value in (*apartment_fields, *building_fields):
            if value is not None:
                facts.append(f"{label}: {value}.")
        for competitor in context.competitors:
            prefix = f"Конкурент {competitor.project_name}"
            facts.append(f"{prefix}, район: {competitor.district}.")
            if competitor.price_per_sqm is not None:
                facts.append(
                    f"{prefix}, цена за м²: {competitor.price_per_sqm}."
                )
            if competitor.advantages is not None:
                facts.append(f"{prefix}, преимущества: {competitor.advantages}.")
            if competitor.disadvantages is not None:
                facts.append(f"{prefix}, недостатки: {competitor.disadvantages}.")
        return "\n".join(f"- {fact}" for fact in facts)

    @staticmethod
    def _unsupported_topics(response: str, factual_source: str) -> list[str]:
        response_text = response.casefold()
        source_text = factual_source.casefold()
        return [
            root
            for root in _FACTUAL_TOPIC_ROOTS
            if root in response_text and root not in source_text
        ]
