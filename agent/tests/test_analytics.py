from unittest.mock import AsyncMock, Mock

import pytest

from app.agents.analytics import (
    ANALYTICS_SYSTEM_PROMPT,
    BACKEND_DATA_ERROR_RESPONSE,
    MISSING_DEAL_RESPONSE,
    AnalyticsAgent,
)
from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.schemas.backend import BackendResponse


DEAL_ID = 101
APARTMENT_ID = 301
BUILDING_ID = 401


def backend_response(data: dict) -> BackendResponse:
    return BackendResponse(
        request_id="backend_req_analytics",
        success=True,
        data=data,
        error=None,
    )


def make_dependencies() -> tuple[Mock, Mock, Mock, Mock, Mock, Mock, Mock]:
    llm = Mock(generate=AsyncMock(return_value="Аналитический ответ"))
    deal_tools = Mock(
        get_deal=AsyncMock(
            return_value=backend_response(
                {
                    "id": DEAL_ID,
                    "client_id": 201,
                    "apartment_id": APARTMENT_ID,
                    "stage": "negotiation",
                    "next_action": "Оценить риски",
                }
            )
        )
    )
    apartment_tools = Mock(
        get_apartment=AsyncMock(
            return_value=backend_response(
                {
                    "id": APARTMENT_ID,
                    "building_id": BUILDING_ID,
                    "number": "142",
                    "floor": 8,
                    "rooms": 2,
                    "area": 64.5,
                    "price": 14200000,
                    "status": "available",
                }
            )
        )
    )
    building_tools = Mock(
        get_building=AsyncMock(
            return_value=backend_response(
                {
                    "id": BUILDING_ID,
                    "name": "ЖК Альфа",
                    "district": "Центральный",
                    "readiness_percent": 76,
                    "planned_delivery": "2027-06-01",
                    "forecast_delivery": "2027-07-15",
                }
            )
        )
    )
    construction_tools = Mock(
        get_construction_events=AsyncMock(
            return_value=backend_response(
                {
                    "events": [
                        {
                            "type": "delivery_delay",
                            "title": "Задержка поставки окон",
                            "risk_level": "medium",
                            "delay_days": 14,
                            "completion_percentage": 70,
                        }
                    ]
                }
            )
        )
    )
    messages_tools = Mock(get_deal_messages=AsyncMock())
    client_tools = Mock(update_client_preferences=AsyncMock())
    return (
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    )


@pytest.mark.asyncio
@pytest.mark.parametrize("intent", ["analyze_risk", "analyze_construction"])
async def test_analytics_workflow_uses_factual_backend_context(intent: str) -> None:
    (
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    ) = make_dependencies()
    agent = AnalyticsAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    )

    result = await agent.generate(
        "Есть ли риски задержки по этому объекту?",
        intent,  # type: ignore[arg-type]
        DEAL_ID,
    )

    assert result == "Аналитический ответ"
    deal_tools.get_deal.assert_awaited_once_with(DEAL_ID)
    apartment_tools.get_apartment.assert_awaited_once_with(APARTMENT_ID)
    building_tools.get_building.assert_awaited_once_with(BUILDING_ID)
    construction_tools.get_construction_events.assert_awaited_once_with(BUILDING_ID)

    user_prompt = llm.generate.await_args.args[0]
    assert f"Intent: {intent}" in user_prompt
    assert "Есть ли риски задержки по этому объекту?" in user_prompt
    assert "ЖК Альфа" in user_prompt
    assert "Задержка поставки окон" in user_prompt
    assert '"risk_level": "medium"' in user_prompt
    assert '"delay_days": 14' in user_prompt
    assert '"completion_percentage": 70' in user_prompt
    assert '"request_id"' not in user_prompt
    assert '"success"' not in user_prompt
    assert '"error"' not in user_prompt
    assert llm.generate.await_args.kwargs["system_prompt"] == ANALYTICS_SYSTEM_PROMPT


@pytest.mark.asyncio
async def test_missing_deal_does_not_call_backend_or_llm() -> None:
    (
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    ) = make_dependencies()
    agent = AnalyticsAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    )

    result = await agent.generate("Оцени риски", "analyze_risk", None)

    assert result == MISSING_DEAL_RESPONSE
    deal_tools.get_deal.assert_not_awaited()
    apartment_tools.get_apartment.assert_not_awaited()
    building_tools.get_building.assert_not_awaited()
    construction_tools.get_construction_events.assert_not_awaited()
    llm.generate.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "backend_error",
    [
        BackendRpcTimeoutError("timeout", code="TIMEOUT"),
        BackendRpcError("not found", code="BUILDING_NOT_FOUND"),
    ],
)
async def test_backend_error_returns_safe_response(backend_error: BackendRpcError) -> None:
    (
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    ) = make_dependencies()
    deal_tools.get_deal.side_effect = backend_error
    agent = AnalyticsAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    )

    result = await agent.generate("Оцени риски", "analyze_risk", DEAL_ID)

    assert result == BACKEND_DATA_ERROR_RESPONSE
    apartment_tools.get_apartment.assert_not_awaited()
    building_tools.get_building.assert_not_awaited()
    construction_tools.get_construction_events.assert_not_awaited()
    llm.generate.assert_not_awaited()


@pytest.mark.asyncio
async def test_invalid_backend_data_returns_safe_response() -> None:
    (
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    ) = make_dependencies()
    deal_tools.get_deal.return_value = backend_response(
        {"id": DEAL_ID, "stage": "negotiation"}
    )
    agent = AnalyticsAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    )

    result = await agent.generate("Оцени риски", "analyze_risk", DEAL_ID)

    assert result == BACKEND_DATA_ERROR_RESPONSE
    apartment_tools.get_apartment.assert_not_awaited()
    llm.generate.assert_not_awaited()


def test_analytics_prompt_forbids_unsupported_factual_claims() -> None:
    prompt = ANALYTICS_SYSTEM_PROMPT.lower()

    assert "запрещено придумывать задержки" in prompt
    assert "даты" in prompt
    assert "причины" in prompt
    assert "risk level" in prompt
    assert "без фактических оснований" in prompt
    assert "completion_percentage" in prompt
    assert "если оно null, не придумывай процент" in prompt
    assert "building.planned_delivery" in prompt
    assert "только когда exact_delivery_date_allowed=true" in prompt


@pytest.mark.asyncio
async def test_exact_delivery_date_is_blocked_when_construction_has_risk() -> None:
    (
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    ) = make_dependencies()
    construction_tools.get_construction_events.return_value = backend_response(
        {"events": [{"type": "supply", "title": "Риск поставки", "risk_level": "medium", "delay_days": 0}]}
    )
    agent = AnalyticsAgent(llm, deal_tools, apartment_tools, building_tools, construction_tools, messages_tools, client_tools)

    await agent.generate("Назови срок", "analyze_risk", DEAL_ID)

    assert '"exact_delivery_date_allowed": false' in llm.generate.await_args.args[0]
    assert '"planned_delivery": null' in llm.generate.await_args.args[0]


@pytest.mark.asyncio
async def test_null_completion_percentage_remains_null_in_analytics_prompt() -> None:
    (
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    ) = make_dependencies()
    construction_tools.get_construction_events.return_value = backend_response(
        {
            "events": [
                {
                    "type": "delivery_delay",
                    "title": "Нет данных о готовности",
                    "risk_level": "high",
                    "delay_days": 25,
                    "completion_percentage": None,
                }
            ]
        }
    )
    agent = AnalyticsAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        construction_tools,
        messages_tools,
        client_tools,
    )

    await agent.generate("Проанализируй строительство", "analyze_construction", DEAL_ID)

    prompt = llm.generate.await_args.args[0]
    assert '"completion_percentage": null' in prompt
