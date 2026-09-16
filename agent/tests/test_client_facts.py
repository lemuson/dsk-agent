import json
from unittest.mock import AsyncMock, Mock

import pytest
from pydantic import ValidationError

from app.agents.analytics import (
    CLIENT_FACTS_ERROR_RESPONSE,
    CLIENT_FACTS_MISSING_DEAL_RESPONSE,
    CLIENT_FACTS_SAVE_ERROR_RESPONSE,
    CLIENT_FACTS_SAVE_SUCCESS_RESPONSE,
    EMPTY_MESSAGES_RESPONSE,
    AnalyticsAgent,
)
from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.schemas.backend import BackendResponse
from app.schemas.client_facts import ClientFacts, DealMessagesData


DEAL_ID = 101
APARTMENT_ID = 301


def backend_response(data: dict) -> BackendResponse:
    return BackendResponse(
        request_id="backend_req_client_facts",
        success=True,
        data=data,
        error=None,
    )


def extracted_facts(**overrides: object) -> ClientFacts:
    values: dict[str, object] = {
        "budget_min": None,
        "budget_max": 15_000_000,
        "rooms": 2,
        "floor_min": 7,
        "floor_max": None,
        "parking_required": True,
        "renovation_required": None,
        "preferred_district": None,
        "purchase_timeline": None,
        "important_factors": [],
        "objections": ["У конкурента дешевле"],
        "summary": "Клиент ищет двухкомнатную квартиру до 15 млн рублей.",
    }
    values.update(overrides)
    return ClientFacts.model_validate(values)


def make_agent(
    *,
    messages: list[dict] | None = None,
    parse_result: ClientFacts | None = None,
) -> tuple[AnalyticsAgent, Mock, Mock, Mock, Mock]:
    llm = Mock(
        generate=AsyncMock(),
        parse_structured=AsyncMock(return_value=parse_result or extracted_facts()),
    )
    deal_tools = Mock(
        get_deal=AsyncMock(
            return_value=backend_response(
                {
                    "id": DEAL_ID,
                    "client_id": 201,
                    "apartment_id": APARTMENT_ID,
                    "stage": "negotiation",
                }
            )
        )
    )
    if messages is None:
        messages = [
            {
                "id": 601,
                "direction": "manager_to_client",
                "body": "Вам нужна трёхкомнатная квартира?",
            },
            {
                "id": 602,
                "direction": "client_to_manager",
                "body": "Нет, нужна двухкомнатная до 15 миллионов.",
            },
            {
                "id": 603,
                "direction": "client_to_manager",
                "body": "Не ниже 7 этажа, парковка обязательна.",
            },
            {
                "id": 604,
                "direction": "client_to_manager",
                "body": "У конкурента дешевле.",
            },
        ]
    messages_tools = Mock(
        get_deal_messages=AsyncMock(
            return_value=backend_response({"messages": messages})
        )
    )
    client_tools = Mock(update_client_preferences=AsyncMock())
    agent = AnalyticsAgent(
        llm,
        deal_tools,
        Mock(),
        Mock(),
        Mock(),
        messages_tools,
        client_tools,
    )
    return agent, llm, deal_tools, messages_tools, client_tools


@pytest.mark.asyncio
async def test_extract_client_facts_uses_deal_messages_and_structured_output() -> None:
    agent, llm, deal_tools, messages_tools, client_tools = make_agent()

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    deal_tools.get_deal.assert_awaited_once_with(DEAL_ID)
    messages_tools.get_deal_messages.assert_awaited_once_with(DEAL_ID)
    llm.parse_structured.assert_awaited_once()
    assert llm.parse_structured.await_args.args[1] is ClientFacts
    llm.generate.assert_not_awaited()
    client_tools.update_client_preferences.assert_awaited_once_with(
        201,
        preferences={
            "rooms": 2,
            "floor_min": 7,
            "parking": True,
            "objections": ["У конкурента дешевле"],
        },
        budget_max=15_000_000,
    )

    prompt = llm.parse_structured.await_args.args[0]
    assert "Нет, нужна двухкомнатная до 15 миллионов." in prompt
    assert "Не ниже 7 этажа, парковка обязательна." in prompt
    assert "У конкурента дешевле." in prompt
    assert "Вам нужна трёхкомнатная квартира?" not in prompt
    assert "15 000 000 ₽" in result
    assert "комнат: 2" in result
    assert "этаж: от 7" in result
    assert "парковка: обязательна" in result
    assert "У конкурента дешевле" in result
    assert CLIENT_FACTS_SAVE_SUCCESS_RESPONSE in result


def test_messages_validate_direction_and_body() -> None:
    valid = DealMessagesData.model_validate(
        {
            "messages": [
                {
                    "id": 602,
                    "direction": "client_to_manager",
                    "body": "Нужна парковка",
                }
            ]
        }
    )
    assert valid.messages[0].direction == "client_to_manager"

    with pytest.raises(ValidationError):
        DealMessagesData.model_validate(
            {
                "messages": [
                    {
                        "id": 602,
                        "direction": "unknown",
                        "body": "Нужна парковка",
                    }
                ]
            }
        )


def test_absent_client_facts_remain_null_or_empty() -> None:
    facts = ClientFacts(
        budget_min=None,
        budget_max=None,
        rooms=None,
        floor_min=None,
        floor_max=None,
        parking_required=None,
        renovation_required=None,
        preferred_district=None,
        purchase_timeline=None,
        important_factors=[],
        objections=[],
        summary="Явных предпочтений не обнаружено.",
    )

    assert facts.budget_min is None
    assert facts.budget_max is None
    assert facts.rooms is None
    assert facts.floor_min is None
    assert facts.floor_max is None
    assert facts.parking_required is None
    assert facts.renovation_required is None
    assert facts.preferred_district is None
    assert facts.purchase_timeline is None
    assert facts.important_factors == []
    assert facts.objections == []

    rendered = AnalyticsAgent._format_client_facts(facts)
    assert rendered == "Краткая сводка: Явных предпочтений не обнаружено."
    assert "бюджет" not in rendered
    assert "комнат" not in rendered
    assert "Возражения клиента" not in rendered


def test_client_facts_structured_schema_requires_every_field() -> None:
    schema = ClientFacts.model_json_schema()

    assert set(schema["required"]) == set(schema["properties"])
    with pytest.raises(ValidationError):
        ClientFacts.model_validate(
            {"summary": "Summary alone must not silently create empty facts."}
        )


@pytest.mark.asyncio
async def test_missing_deal_does_not_call_tools_or_llm() -> None:
    agent, llm, deal_tools, messages_tools, client_tools = make_agent()

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        None,
    )

    assert result == CLIENT_FACTS_MISSING_DEAL_RESPONSE
    deal_tools.get_deal.assert_not_awaited()
    messages_tools.get_deal_messages.assert_not_awaited()
    llm.parse_structured.assert_not_awaited()
    client_tools.update_client_preferences.assert_not_awaited()


@pytest.mark.asyncio
async def test_empty_client_message_history_returns_safe_response() -> None:
    manager_message = {
        "id": 601,
        "direction": "manager_to_client",
        "body": "Какой бюджет рассматриваете?",
    }
    agent, llm, _, _, client_tools = make_agent(messages=[manager_message])

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    assert result == EMPTY_MESSAGES_RESPONSE
    llm.parse_structured.assert_not_awaited()
    client_tools.update_client_preferences.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "backend_error",
    [
        BackendRpcTimeoutError("timeout", code="TIMEOUT"),
        BackendRpcError("not found", code="DEAL_NOT_FOUND"),
    ],
)
async def test_backend_error_returns_safe_response(backend_error: BackendRpcError) -> None:
    agent, llm, deal_tools, messages_tools, client_tools = make_agent()
    deal_tools.get_deal.side_effect = backend_error

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    assert result == CLIENT_FACTS_ERROR_RESPONSE
    messages_tools.get_deal_messages.assert_not_awaited()
    llm.parse_structured.assert_not_awaited()
    client_tools.update_client_preferences.assert_not_awaited()


@pytest.mark.asyncio
async def test_invalid_messages_response_returns_safe_response() -> None:
    agent, llm, _, messages_tools, client_tools = make_agent()
    messages_tools.get_deal_messages.return_value = backend_response(
        {
            "messages": [
                {
                    "id": "not-a-uuid",
                    "direction": "client_to_manager",
                    "body": "Текст",
                }
            ]
        }
    )

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    assert result == CLIENT_FACTS_ERROR_RESPONSE
    llm.parse_structured.assert_not_awaited()
    client_tools.update_client_preferences.assert_not_awaited()


@pytest.mark.asyncio
async def test_primary_json_decode_error_and_invalid_fallback_return_safe_response() -> None:
    agent, llm, _, _, client_tools = make_agent()
    llm.parse_structured.side_effect = json.JSONDecodeError("invalid", "x", 0)
    llm.generate.return_value = "not valid JSON"

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    assert result == CLIENT_FACTS_ERROR_RESPONSE
    llm.parse_structured.assert_awaited_once()
    llm.generate.assert_awaited_once()
    client_tools.update_client_preferences.assert_not_awaited()


@pytest.mark.asyncio
async def test_primary_json_decode_error_uses_successful_json_fallback() -> None:
    agent, llm, _, _, client_tools = make_agent()
    facts = extracted_facts()
    llm.parse_structured.side_effect = json.JSONDecodeError("invalid", "x", 0)
    llm.generate.return_value = f"  \n{facts.model_dump_json()}\n  "

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    assert "комнат: 2" in result
    llm.parse_structured.assert_awaited_once()
    llm.generate.assert_awaited_once()
    client_tools.update_client_preferences.assert_awaited_once()


@pytest.mark.asyncio
async def test_fallback_valid_json_with_invalid_client_facts_returns_error() -> None:
    agent, llm, _, _, client_tools = make_agent()
    llm.parse_structured.side_effect = ValueError("invalid structured response")
    llm.generate.return_value = json.dumps(
        {"summary": "Остальные обязательные structured fields отсутствуют."}
    )

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    assert result == CLIENT_FACTS_ERROR_RESPONSE
    client_tools.update_client_preferences.assert_not_awaited()


@pytest.mark.asyncio
async def test_successful_fallback_preserves_fields_and_updates_preferences() -> None:
    agent, llm, _, _, client_tools = make_agent()
    facts = extracted_facts()
    llm.parse_structured.side_effect = ValueError("invalid structured response")
    llm.generate.return_value = facts.model_dump_json()

    result = await agent.analyze_dialog(DEAL_ID)

    assert result.analysis.model_dump() == facts.model_dump()
    assert result.preferences_updated is True
    client_tools.update_client_preferences.assert_awaited_once_with(
        201,
        preferences={
            "rooms": 2,
            "floor_min": 7,
            "parking": True,
            "objections": ["У конкурента дешевле"],
        },
        budget_max=15_000_000,
    )


@pytest.mark.asyncio
async def test_update_error_preserves_extracted_summary() -> None:
    agent, _, _, _, client_tools = make_agent()
    client_tools.update_client_preferences.side_effect = BackendRpcError(
        "backend unavailable",
        code="BACKEND_UNAVAILABLE",
    )

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    assert "Клиент ищет двухкомнатную квартиру до 15 млн рублей." in result
    assert "15 000 000 ₽" in result
    assert CLIENT_FACTS_SAVE_ERROR_RESPONSE in result
    assert CLIENT_FACTS_SAVE_SUCCESS_RESPONSE not in result


@pytest.mark.asyncio
async def test_empty_extracted_facts_are_not_sent_to_backend() -> None:
    empty_facts = ClientFacts(
        budget_min=None,
        budget_max=None,
        rooms=None,
        floor_min=None,
        floor_max=None,
        parking_required=None,
        renovation_required=None,
        preferred_district=None,
        purchase_timeline=None,
        important_factors=[],
        objections=[],
        summary="Явных предпочтений не обнаружено.",
    )
    agent, _, _, _, client_tools = make_agent(parse_result=empty_facts)

    result = await agent.generate(
        "Проанализируй переписку",
        "extract_client_facts",
        DEAL_ID,
    )

    assert result == "Краткая сводка: Явных предпочтений не обнаружено."
    client_tools.update_client_preferences.assert_not_awaited()
    assert CLIENT_FACTS_SAVE_SUCCESS_RESPONSE not in result
