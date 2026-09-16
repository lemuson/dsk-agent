from unittest.mock import AsyncMock, Mock

import pytest
from pydantic import ValidationError

from app.agents.router import ROUTER_PROMPT, AgentRouter, RouteDecision


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("message", "agent", "intent"),
    [
        (
            "Клиент говорит, что у конкурента дешевле",
            "negotiation",
            "handle_objection",
        ),
        ("Сформируй коммерческое предложение", "offer", "create_offer"),
        (
            "Есть ли риски задержки по этому объекту?",
            "analytics",
            "analyze_risk",
        ),
        ("Привет", "general", "general_chat"),
    ],
)
async def test_router_returns_structured_decision(
    message: str,
    agent: str,
    intent: str,
) -> None:
    expected = RouteDecision.model_validate(
        {"agent": agent, "intent": intent, "confidence": 0.9}
    )
    llm = Mock(parse_structured=AsyncMock(return_value=expected))
    router = AgentRouter(llm)

    result = await router.route(message)

    assert result.agent == agent
    assert result.intent == intent
    prompt = llm.parse_structured.await_args.args[0]
    assert message in prompt
    assert llm.parse_structured.await_args.args[1] is RouteDecision


@pytest.mark.asyncio
async def test_invalid_structured_responses_fall_back_after_one_retry() -> None:
    llm = Mock(
        parse_structured=AsyncMock(
            side_effect=[ValueError("invalid JSON"), ValueError("invalid schema")]
        )
    )
    router = AgentRouter(llm)

    result = await router.route("Неоднозначный запрос")

    assert result == RouteDecision(
        agent="general",
        intent="unknown",
        confidence=0.0,
    )
    assert llm.parse_structured.await_count == 2


@pytest.mark.parametrize("confidence", [-0.01, 1.01])
def test_route_confidence_must_be_between_zero_and_one(confidence: float) -> None:
    with pytest.raises(ValidationError):
        RouteDecision(
            agent="general",
            intent="general_chat",
            confidence=confidence,
        )


def test_route_intent_must_match_agent() -> None:
    with pytest.raises(ValidationError):
        RouteDecision(
            agent="analytics",
            intent="create_offer",
            confidence=0.5,
        )


@pytest.mark.parametrize(
    "message",
    [
        "Сформируй коммерческое предложение",
        "Сделай КП со скидкой 3%",
        "Подготовь предложение со скидкой 7 процентов",
        "Хочу предложить клиенту скидку 10%",
    ],
)
def test_router_prompt_contains_offer_routing_examples(message: str) -> None:
    assert message in ROUTER_PROMPT
