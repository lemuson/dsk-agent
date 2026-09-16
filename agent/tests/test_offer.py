from decimal import Decimal
from unittest.mock import AsyncMock, Mock

import pytest

from app.agents.offer import (
    BACKEND_DATA_ERROR_RESPONSE,
    DISCOUNT_CLARIFICATION_RESPONSE,
    DISCOUNT_PRECISION_RESPONSE,
    DISCOUNT_RANGE_RESPONSE,
    MISSING_DEAL_RESPONSE,
    OFFER_SAVE_ERROR_RESPONSE,
    OFFER_SYSTEM_PROMPT,
    OfferAgent,
)
from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.graphs.offer_graph import APPROVAL_REQUEST_ERROR_RESPONSE
from app.schemas.backend import BackendResponse
from app.schemas.offer import OfferRequestFacts


DEAL_ID = 101
USER_ID = 15
CLIENT_ID = 201
APARTMENT_ID = 301


def backend_response(data: dict) -> BackendResponse:
    return BackendResponse(
        request_id="backend_req_offer",
        success=True,
        data=data,
        error=None,
    )


def calculation_data(*, requires_approval: bool = False) -> dict:
    return {
        "deal_id": DEAL_ID,
        "base_price": 14_200_000,
        "discount_percent": 3,
        "discount_amount": 426_000,
        "final_price": 13_774_000,
        "max_allowed_discount": 3,
        "requires_approval": requires_approval,
    }


def make_dependencies() -> tuple[Mock, Mock, Mock, Mock, Mock]:
    llm = Mock(
        generate=AsyncMock(return_value="Персональный текст КП"),
        parse_structured=AsyncMock(
            return_value=OfferRequestFacts(
                discount_mentioned=False,
                requested_discount_percent=None,
            )
        ),
    )
    deal_tools = Mock(
        get_deal=AsyncMock(
            return_value=backend_response(
                {
                    "id": DEAL_ID,
                    "client_id": CLIENT_ID,
                    "apartment_id": APARTMENT_ID,
                    "stage": "negotiation",
                }
            )
        )
    )
    client_tools = Mock(
        get_client=AsyncMock(
            return_value=backend_response(
                {
                    "id": CLIENT_ID,
                    "full_name": "Иван Иванов",
                    "budget_max": 15_000_000,
                    "preferences": {"rooms": 2, "parking": True},
                }
            )
        )
    )
    apartment_tools = Mock(
        get_apartment=AsyncMock(
            return_value=backend_response(
                {
                    "id": APARTMENT_ID,
                    "building_id": 401,
                    "number": "142",
                    "floor": 8,
                    "rooms": 2,
                    "area": 64.5,
                    "price": 14_200_000,
                    "status": "available",
                }
            )
        )
    )
    offer_tools = Mock(
        calculate_offer=AsyncMock(return_value=backend_response(calculation_data())),
        create_offer=AsyncMock(
            return_value=backend_response(
                {
                    "offer_id": 501,
                    "status": "approved",
                }
            )
        ),
        request_offer_approval=AsyncMock(
            return_value=backend_response(
                {
                    "offer_id": 501,
                    "status": "pending_approval",
                }
            )
        ),
    )
    return llm, deal_tools, client_tools, apartment_tools, offer_tools


def set_extracted_discount(
    llm: Mock,
    value: str | None,
    *,
    mentioned: bool = True,
) -> None:
    llm.parse_structured.return_value = OfferRequestFacts(
        discount_mentioned=mentioned,
        requested_discount_percent=Decimal(value) if value is not None else None,
    )


@pytest.mark.asyncio
async def test_create_offer_uses_factual_backend_workflow() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate("Сформируй КП", "create_offer", DEAL_ID, USER_ID)

    deal_tools.get_deal.assert_awaited_once_with(DEAL_ID)
    client_tools.get_client.assert_awaited_once_with(CLIENT_ID)
    apartment_tools.get_apartment.assert_awaited_once_with(APARTMENT_ID)
    offer_tools.calculate_offer.assert_awaited_once_with(DEAL_ID, USER_ID, 0)
    offer_tools.create_offer.assert_awaited_once_with(
        DEAL_ID,
        USER_ID,
        3.0,
        "Персональный текст КП",
    )

    prompt = llm.generate.await_args.args[0]
    assert "Иван Иванов" in prompt
    assert '"number": "142"' in prompt
    assert '"base_price": 14200000' in prompt
    assert '"discount_amount": 426000' in prompt
    assert '"final_price": 13774000' in prompt
    assert '"request_id"' not in prompt
    assert '"success"' not in prompt
    assert '"error"' not in prompt
    assert llm.generate.await_args.kwargs["system_prompt"] == OFFER_SYSTEM_PROMPT

    assert "Персональный текст КП" in result
    assert "14 200 000 ₽" in result
    assert "13 774 000 ₽" in result
    assert "Предложение сохранено." in result
    offer_tools.request_offer_approval.assert_not_awaited()


@pytest.mark.asyncio
async def test_create_offer_preserves_selected_parking_and_storage() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    offer_tools.calculate_offer.return_value = backend_response(
        {
            **calculation_data(),
            "base_price": 15_930_000,
            "apartment_price": 14_200_000,
            "parking_unit_id": 41,
            "parking_number": "P-041",
            "parking_price": 1_250_000,
            "storage_unit_id": 42,
            "storage_number": "K-018",
            "storage_price": 480_000,
            "discount_amount": 477_900,
            "final_price": 15_452_100,
        }
    )
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate(
        "Сформируй КП",
        "create_offer",
        DEAL_ID,
        USER_ID,
        parking_unit_id=41,
        storage_unit_id=42,
    )

    offer_tools.calculate_offer.assert_awaited_once_with(
        DEAL_ID,
        USER_ID,
        0,
        parking_unit_id=41,
        storage_unit_id=42,
    )
    offer_tools.create_offer.assert_awaited_once_with(
        DEAL_ID,
        USER_ID,
        3.0,
        "Персональный текст КП",
        parking_unit_id=41,
        storage_unit_id=42,
    )
    assert "Парковка P-041: 1 250 000 ₽" in result
    assert "Кладовая K-018: 480 000 ₽" in result
    assert "Стоимость до скидки: 15 930 000 ₽" in result


@pytest.mark.asyncio
async def test_requires_approval_creates_offer_and_requests_approval() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    offer_tools.calculate_offer.return_value = backend_response(
        calculation_data(requires_approval=True)
    )
    offer_tools.create_offer.return_value = backend_response(
        {
            "offer_id": 501,
            "status": "draft",
        }
    )
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate("Сформируй КП", "create_offer", DEAL_ID, USER_ID)

    assert "Персональный текст КП" in result
    assert "Запрос на согласование отправлен" in result
    llm.generate.assert_awaited_once()
    offer_tools.create_offer.assert_awaited_once()
    offer_tools.request_offer_approval.assert_awaited_once_with(501, USER_ID)

    state = await agent._graph.run(
        {
            "user_id": USER_ID,
            "deal_id": DEAL_ID,
            "message": "Сформируй КП",
            "intent": "create_offer",
            "requested_discount_percent": 5,
            "approval_required": False,
        }
    )
    assert state["approval_status"] == "waiting"
    assert state["offer_id"] == 501


@pytest.mark.asyncio
async def test_calculate_offer_returns_backend_values_without_creating_offer() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate("Рассчитай КП", "calculate_offer", DEAL_ID, USER_ID)

    assert "Базовая стоимость: 14 200 000 ₽" in result
    assert "Скидка: 3%" in result
    assert "Итоговая стоимость: 13 774 000 ₽" in result
    llm.generate.assert_not_awaited()
    offer_tools.create_offer.assert_not_awaited()
    offer_tools.request_offer_approval.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("message", "structured_value", "expected"),
    [
        ("Сформируй предложение со скидкой 7%", "7", 7),
        ("Сделай КП со скидкой 3.5%", "3.5", 3.5),
        ("Предложи клиенту скидку 10 процентов", "10", 10),
        ("Сформируй КП со скидкой 5,5%", "5.5", 5.5),
        ("Сформируй КП со скидкой 0%", "0", 0),
    ],
)
async def test_explicit_discount_is_extracted_and_sent_to_backend(
    message: str,
    structured_value: str,
    expected: int | float,
) -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    set_extracted_discount(llm, structured_value)
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    await agent.generate(message, "calculate_offer", DEAL_ID, USER_ID)

    llm.parse_structured.assert_awaited_once()
    assert llm.parse_structured.await_args.args[1] is OfferRequestFacts
    offer_tools.calculate_offer.assert_awaited_once_with(DEAL_ID, USER_ID, expected)


@pytest.mark.asyncio
async def test_no_discount_uses_zero_without_clarification() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate(
        "Сформируй коммерческое предложение",
        "calculate_offer",
        DEAL_ID,
        USER_ID,
    )

    assert result != DISCOUNT_CLARIFICATION_RESPONSE
    offer_tools.calculate_offer.assert_awaited_once_with(DEAL_ID, USER_ID, 0)


@pytest.mark.asyncio
async def test_ambiguous_discount_requests_clarification_without_backend_calls() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    set_extracted_discount(llm, None)
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate(
        "Дай максимальную скидку",
        "create_offer",
        DEAL_ID,
        USER_ID,
        42,
    )

    assert result == DISCOUNT_CLARIFICATION_RESPONSE
    deal_tools.get_deal.assert_not_awaited()
    offer_tools.calculate_offer.assert_not_awaited()
    offer_tools.create_offer.assert_not_awaited()


@pytest.mark.asyncio
async def test_llm_cannot_invent_number_for_ambiguous_discount() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    set_extracted_discount(llm, "7")
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate(
        "Сделай хорошую скидку",
        "create_offer",
        DEAL_ID,
        USER_ID,
    )

    assert result == DISCOUNT_CLARIFICATION_RESPONSE
    offer_tools.calculate_offer.assert_not_awaited()
    offer_tools.create_offer.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("message", "expected_response"),
    [
        ("Сделай КП со скидкой -1%", DISCOUNT_RANGE_RESPONSE),
        ("Сделай КП со скидкой 101%", DISCOUNT_RANGE_RESPONSE),
        ("Сделай КП со скидкой 5.555%", DISCOUNT_PRECISION_RESPONSE),
    ],
)
async def test_invalid_discount_does_not_start_backend_workflow(
    message: str,
    expected_response: str,
) -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate(
        message,
        "create_offer",
        DEAL_ID,
        USER_ID,
        42,
    )

    assert result == expected_response
    deal_tools.get_deal.assert_not_awaited()
    offer_tools.calculate_offer.assert_not_awaited()
    offer_tools.create_offer.assert_not_awaited()


@pytest.mark.asyncio
async def test_pending_offer_continues_with_discount_in_same_session() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    llm.parse_structured.side_effect = [
        OfferRequestFacts(
            discount_mentioned=True,
            requested_discount_percent=None,
        ),
        OfferRequestFacts(
            discount_mentioned=True,
            requested_discount_percent=Decimal("7"),
        ),
    ]
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    first = await agent.generate(
        "Сделай предложение с хорошей скидкой",
        "create_offer",
        DEAL_ID,
        USER_ID,
        42,
    )
    second = await agent.resume_pending("7%", USER_ID, 42)

    assert first == DISCOUNT_CLARIFICATION_RESPONSE
    assert second is not None
    assert second[1] == "create_offer"
    offer_tools.calculate_offer.assert_awaited_once_with(DEAL_ID, USER_ID, 7)
    generation_prompt = llm.generate.await_args.args[0]
    assert "Сделай предложение с хорошей скидкой" in generation_prompt
    assert "Уточнённый процент скидки: 7%" in generation_prompt
    assert await agent.resume_pending("7%", USER_ID, 42) is None


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("discount", "requires_approval", "create_status", "approval_called"),
    [
        (3, False, "approved", False),
        (7, True, "draft", True),
    ],
)
async def test_approval_branch_uses_only_backend_requires_approval(
    discount: int,
    requires_approval: bool,
    create_status: str,
    approval_called: bool,
) -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    set_extracted_discount(llm, str(discount))
    offer_tools.calculate_offer.return_value = backend_response(
        {
            **calculation_data(requires_approval=requires_approval),
            "discount_percent": discount,
        }
    )
    offer_tools.create_offer.return_value = backend_response(
        {"offer_id": 501, "status": create_status}
    )
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate(
        f"Сделай КП со скидкой {discount}%",
        "create_offer",
        DEAL_ID,
        USER_ID,
    )

    offer_tools.calculate_offer.assert_awaited_once_with(DEAL_ID, USER_ID, discount)
    if approval_called:
        offer_tools.request_offer_approval.assert_awaited_once_with(501, USER_ID)
        assert "требуется согласование" in result
        assert "Запрос на согласование отправлен" in result
    else:
        offer_tools.request_offer_approval.assert_not_awaited()
        assert "Предложение сохранено." in result


@pytest.mark.parametrize("value", ["-1", "101", "5.555"])
def test_offer_request_facts_validates_discount_bounds_and_precision(value: str) -> None:
    with pytest.raises(ValueError):
        OfferRequestFacts(
            discount_mentioned=True,
            requested_discount_percent=Decimal(value),
        )


@pytest.mark.asyncio
async def test_missing_deal_does_not_call_tools_or_llm() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate("Сформируй КП", "create_offer", None, USER_ID)

    assert result == MISSING_DEAL_RESPONSE
    deal_tools.get_deal.assert_not_awaited()
    client_tools.get_client.assert_not_awaited()
    apartment_tools.get_apartment.assert_not_awaited()
    offer_tools.calculate_offer.assert_not_awaited()
    offer_tools.create_offer.assert_not_awaited()
    llm.generate.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("dependency", "error"),
    [
        ("deal", BackendRpcTimeoutError("timeout", code="TIMEOUT")),
        ("deal", BackendRpcError("deal missing", code="DEAL_NOT_FOUND")),
        ("client", BackendRpcError("client missing", code="CLIENT_NOT_FOUND")),
        (
            "apartment",
            BackendRpcError("apartment missing", code="APARTMENT_NOT_FOUND"),
        ),
    ],
)
async def test_backend_read_error_returns_safe_response(
    dependency: str,
    error: BackendRpcError,
) -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    failing_method = {
        "deal": deal_tools.get_deal,
        "client": client_tools.get_client,
        "apartment": apartment_tools.get_apartment,
    }[dependency]
    failing_method.side_effect = error
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate("Сформируй КП", "create_offer", DEAL_ID, USER_ID)

    assert result == BACKEND_DATA_ERROR_RESPONSE
    llm.generate.assert_not_awaited()
    offer_tools.create_offer.assert_not_awaited()


@pytest.mark.asyncio
async def test_create_error_preserves_generated_text() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    offer_tools.create_offer.side_effect = BackendRpcError(
        "create failed",
        code="CREATE_FAILED",
    )
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate("Сформируй КП", "create_offer", DEAL_ID, USER_ID)

    assert "Персональный текст КП" in result
    assert "14 200 000 ₽" in result
    assert OFFER_SAVE_ERROR_RESPONSE in result
    assert "Предложение сохранено." not in result


@pytest.mark.asyncio
@pytest.mark.parametrize("operation", ["calculate", "create", "request_approval"])
async def test_forbidden_offer_operation_is_not_hidden_as_success(
    operation: str,
) -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    if operation == "calculate":
        offer_tools.calculate_offer.side_effect = BackendRpcError(
            "forbidden", code="FORBIDDEN"
        )
    elif operation == "create":
        offer_tools.create_offer.side_effect = BackendRpcError(
            "forbidden", code="FORBIDDEN"
        )
    else:
        offer_tools.calculate_offer.return_value = backend_response(
            calculation_data(requires_approval=True)
        )
        offer_tools.create_offer.return_value = backend_response(
            {"offer_id": 501, "status": "draft"}
        )
        offer_tools.request_offer_approval.side_effect = BackendRpcError(
            "forbidden", code="FORBIDDEN"
        )
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    with pytest.raises(BackendRpcError, match="forbidden") as captured:
        await agent.generate("Сформируй КП", "create_offer", DEAL_ID, USER_ID)

    assert captured.value.code == "FORBIDDEN"


@pytest.mark.asyncio
async def test_request_approval_error_preserves_offer_and_generated_text() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    offer_tools.calculate_offer.return_value = backend_response(
        calculation_data(requires_approval=True)
    )
    offer_tools.create_offer.return_value = backend_response(
        {
            "offer_id": 501,
            "status": "draft",
        }
    )
    offer_tools.request_offer_approval.side_effect = BackendRpcError(
        "approval failed",
        code="APPROVAL_FAILED",
    )
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    state = await agent._graph.run(
        {
            "user_id": USER_ID,
            "deal_id": DEAL_ID,
            "message": "Сформируй КП",
            "intent": "create_offer",
            "requested_discount_percent": 5,
            "approval_required": False,
        }
    )

    assert state["offer_id"] == 501
    assert state["error"] == "APPROVAL_REQUEST_ERROR"
    assert "Персональный текст КП" in state["result_message"]
    assert APPROVAL_REQUEST_ERROR_RESPONSE in state["result_message"]


@pytest.mark.asyncio
async def test_gigachat_error_is_controlled_graph_state() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    llm.generate.side_effect = RuntimeError("LLM unavailable")
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    result = await agent.generate("Сформируй КП", "create_offer", DEAL_ID, USER_ID)

    assert result == "Не удалось подготовить текст коммерческого предложения."
    offer_tools.create_offer.assert_not_awaited()


@pytest.mark.asyncio
async def test_offer_graph_is_compiled_once() -> None:
    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)
    compiled = agent._graph.compiled

    await agent.generate("Сформируй КП", "create_offer", DEAL_ID, USER_ID)
    await agent.generate("Сформируй ещё раз", "create_offer", DEAL_ID, USER_ID)

    assert agent._graph.compiled is compiled


@pytest.mark.asyncio
async def test_parallel_offer_graph_executions_keep_state_separate() -> None:
    import asyncio

    llm, deal_tools, client_tools, apartment_tools, offer_tools = make_dependencies()

    async def generate_for_prompt(prompt: str, **_: object) -> str:
        return "Текст A" if "Запрос A" in prompt else "Текст B"

    llm.generate.side_effect = generate_for_prompt
    agent = OfferAgent(llm, deal_tools, client_tools, apartment_tools, offer_tools)

    first, second = await asyncio.gather(
        agent.generate("Запрос A", "create_offer", DEAL_ID, USER_ID),
        agent.generate("Запрос B", "create_offer", DEAL_ID, USER_ID),
    )

    assert "Текст A" in first and "Текст B" not in first
    assert "Текст B" in second and "Текст A" not in second


def test_offer_prompt_forbids_financial_and_factual_invention() -> None:
    prompt = OFFER_SYSTEM_PROMPT.lower()

    assert "запрещено пересчитывать цену" in prompt
    assert "менять discount" in prompt
    assert "придумывать стоимость" in prompt
    assert "характеристики квартиры" in prompt
    assert "данные клиента" in prompt
