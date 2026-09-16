from unittest.mock import AsyncMock, Mock

import pytest

from app.agents.negotiation import (
    BACKEND_DATA_ERROR_RESPONSE,
    MISSING_DEAL_RESPONSE,
    NEGOTIATION_SYSTEM_PROMPT,
    NegotiationAgent,
)
from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.schemas.backend import BackendResponse

DEAL_ID = 101
APARTMENT_ID = 301
BUILDING_ID = 401


def backend_response(data: dict) -> BackendResponse:
    return BackendResponse(
        request_id="backend_req_negotiation",
        success=True,
        data=data,
        error=None,
    )


def make_dependencies() -> tuple[Mock, Mock, Mock, Mock, Mock]:
    llm = Mock(generate=AsyncMock(return_value="Фактический контраргумент"))
    deal_tools = Mock(
        get_deal=AsyncMock(
            return_value=backend_response(
                {
                    "id": DEAL_ID,
                    "client_id": 201,
                    "apartment_id": APARTMENT_ID,
                    "stage": "negotiation",
                    "next_action": "Подготовить контраргумент",
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
    competitor_tools = Mock(
        list_competitors=AsyncMock(
            return_value=backend_response(
                {
                    "competitors": [
                        {
                            "project_name": "ЖК Конкурент",
                            "district": "Центральный",
                            "price_per_sqm": 210000,
                            "advantages": "Более низкая цена",
                            "disadvantages": "Более поздний срок сдачи",
                        }
                    ]
                }
            )
        )
    )
    return llm, deal_tools, apartment_tools, building_tools, competitor_tools


def test_negotiation_prompt_blocks_exact_delivery_without_green_zone() -> None:
    prompt = NEGOTIATION_SYSTEM_PROMPT.lower()
    assert "точную дату" in prompt
    assert "зелёной зоны" in prompt


@pytest.mark.asyncio
@pytest.mark.parametrize("intent", ["handle_objection", "compare_competitor"])
async def test_negotiation_workflow_uses_factual_backend_context(intent: str) -> None:
    llm, deal_tools, apartment_tools, building_tools, competitor_tools = (
        make_dependencies()
    )
    agent = NegotiationAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        competitor_tools,
    )

    result = await agent.generate(
        "Клиент говорит, что у конкурента дешевле",
        intent,  # type: ignore[arg-type]
        DEAL_ID,
    )

    assert result == "Фактический контраргумент"
    deal_tools.get_deal.assert_awaited_once_with(DEAL_ID)
    apartment_tools.get_apartment.assert_awaited_once_with(APARTMENT_ID)
    building_tools.get_building.assert_awaited_once_with(BUILDING_ID)
    competitor_tools.list_competitors.assert_awaited_once_with("Центральный")

    user_prompt = llm.generate.await_args.args[0]
    assert f"Intent: {intent}" in user_prompt
    assert "Клиент говорит, что у конкурента дешевле" in user_prompt
    assert "ЖК Альфа" in user_prompt
    assert "ЖК Конкурент" in user_prompt
    assert "14200000" in user_prompt
    assert '"request_id"' not in user_prompt
    assert '"success"' not in user_prompt
    assert '"error"' not in user_prompt
    assert llm.generate.await_args.kwargs["system_prompt"] == NEGOTIATION_SYSTEM_PROMPT


@pytest.mark.asyncio
async def test_missing_deal_does_not_call_backend_or_llm() -> None:
    llm, deal_tools, apartment_tools, building_tools, competitor_tools = (
        make_dependencies()
    )
    agent = NegotiationAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        competitor_tools,
    )

    result = await agent.generate("Возражение клиента", "handle_objection", None)

    assert result == MISSING_DEAL_RESPONSE
    deal_tools.get_deal.assert_not_awaited()
    apartment_tools.get_apartment.assert_not_awaited()
    building_tools.get_building.assert_not_awaited()
    competitor_tools.list_competitors.assert_not_awaited()
    llm.generate.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "backend_error",
    [
        BackendRpcTimeoutError("timeout", code="TIMEOUT"),
        BackendRpcError("not found", code="DEAL_NOT_FOUND"),
    ],
)
async def test_backend_error_returns_safe_response(backend_error: BackendRpcError) -> None:
    llm, deal_tools, apartment_tools, building_tools, competitor_tools = (
        make_dependencies()
    )
    deal_tools.get_deal.side_effect = backend_error
    agent = NegotiationAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        competitor_tools,
    )

    result = await agent.generate("Возражение клиента", "handle_objection", DEAL_ID)

    assert result == BACKEND_DATA_ERROR_RESPONSE
    apartment_tools.get_apartment.assert_not_awaited()
    building_tools.get_building.assert_not_awaited()
    competitor_tools.list_competitors.assert_not_awaited()
    llm.generate.assert_not_awaited()


@pytest.mark.asyncio
async def test_invalid_backend_data_returns_safe_response() -> None:
    llm, deal_tools, apartment_tools, building_tools, competitor_tools = (
        make_dependencies()
    )
    deal_tools.get_deal.return_value = backend_response(
        {"id": DEAL_ID, "stage": "negotiation"}
    )
    agent = NegotiationAgent(
        llm,
        deal_tools,
        apartment_tools,
        building_tools,
        competitor_tools,
    )

    result = await agent.generate("Возражение клиента", "handle_objection", DEAL_ID)

    assert result == BACKEND_DATA_ERROR_RESPONSE
    apartment_tools.get_apartment.assert_not_awaited()
    llm.generate.assert_not_awaited()


def test_negotiation_prompt_forbids_unsupported_factual_claims() -> None:
    prompt = NEGOTIATION_SYSTEM_PROMPT.lower()

    assert "запрещено придумывать цены" in prompt
    assert "запрещено придумывать сроки строительства" in prompt
    assert "запрещено придумывать инфраструктуру" in prompt
    assert "недостатки" in prompt
    assert "конкурента" in prompt
    assert "фактическом контексте" in prompt
    assert "если преимущество не подтверждено" in prompt
    assert "варианты отделки" in prompt
    assert "разнообразие планировок" in prompt
    assert "инфраструктуру, транспорт, ипотеку, скидки" in prompt
