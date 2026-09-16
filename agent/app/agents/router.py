import json
import logging
from typing import Literal, Protocol

from pydantic import BaseModel, ConfigDict, Field, ValidationError, model_validator

logger = logging.getLogger(__name__)

AgentName = Literal["analytics", "negotiation", "offer", "general"]
IntentName = Literal[
    "analyze_risk",
    "analyze_construction",
    "extract_client_facts",
    "handle_objection",
    "compare_competitor",
    "create_offer",
    "calculate_offer",
    "offer_status",
    "general_chat",
    "unknown",
]

INTENTS_BY_AGENT: dict[str, set[str]] = {
    "analytics": {"analyze_risk", "analyze_construction", "extract_client_facts"},
    "negotiation": {"handle_objection", "compare_competitor"},
    "offer": {"create_offer", "calculate_offer", "offer_status"},
    "general": {"general_chat", "unknown"},
}

ROUTER_PROMPT = """Ты маршрутизатор запросов менеджера по продажам недвижимости.
Классифицируй сообщение и верни structured JSON по переданной схеме.

Допустимые маршруты:
- analytics: analyze_risk, analyze_construction, extract_client_facts
- negotiation: handle_objection, compare_competitor
- offer: create_offer, calculate_offer, offer_status
- general: general_chat, unknown

Правила:
- Выбирай ровно один agent и соответствующий ему intent.
- confidence — число от 0 до 1.
- Не отвечай на сообщение и не выполняй никаких действий.
- Если запрос нельзя уверенно классифицировать, выбери general / unknown.

Примеры для offer / create_offer:
- «Сформируй коммерческое предложение»
- «Сделай КП со скидкой 3%»
- «Подготовь предложение со скидкой 7 процентов»
- «Хочу предложить клиенту скидку 10%»

Сообщение пользователя:
{message}
"""


class RouteDecision(BaseModel):
    model_config = ConfigDict(extra="forbid")

    agent: AgentName
    intent: IntentName
    confidence: float = Field(ge=0.0, le=1.0)

    @model_validator(mode="after")
    def validate_agent_intent_pair(self) -> "RouteDecision":
        if self.intent not in INTENTS_BY_AGENT[self.agent]:
            raise ValueError(
                f"intent {self.intent!r} is not valid for agent {self.agent!r}"
            )
        return self


class StructuredOutputClient(Protocol):
    async def parse_structured(
        self,
        prompt: str,
        response_model: type[RouteDecision],
    ) -> RouteDecision: ...


class AgentRouter:
    def __init__(self, llm: StructuredOutputClient) -> None:
        self._llm = llm

    async def route(self, message: str) -> RouteDecision:
        prompt = ROUTER_PROMPT.format(message=message)

        for attempt in range(1, 3):
            try:
                decision = await self._llm.parse_structured(prompt, RouteDecision)
                logger.info(
                    "Chat route selected: agent=%s intent=%s confidence=%.3f",
                    decision.agent,
                    decision.intent,
                    decision.confidence,
                )
                return decision
            except (json.JSONDecodeError, ValidationError, ValueError) as exc:
                logger.warning(
                    "Invalid structured route response: attempt=%d/2 error_type=%s",
                    attempt,
                    type(exc).__name__,
                )

        logger.error("Structured route fallback selected after 2 invalid responses")
        return RouteDecision(agent="general", intent="unknown", confidence=0.0)
