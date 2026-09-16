import json
from unittest.mock import AsyncMock, Mock

import httpx
import pytest
from pydantic import ValidationError

from app.agents.analytics import (
    AnalyticsAgent,
    ClientFactsExtractionError,
    NoClientMessagesError,
)
from app.agents.dialog import DialogCommandError, DialogService
from app.agents.negotiation import NegotiationAgent
from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.broker.consumer import AgentConsumer
from app.broker.connection import DIALOG_ROUTING_KEYS, RabbitMQConnection
from app.schemas.backend import BackendResponse
from app.schemas.client_facts import ClientFacts
from app.schemas.dialog import (
    DialogAnalyzeRequest,
    DialogAnalyzeResult,
    DialogReplyAssistRequest,
    DialogReplyAssistResult,
    ReplyAssistAnalysis,
)
from app.schemas.negotiation import DealFacts


REQUEST_ID = "req_test_dialog"
USER_ID = 15
DEAL_ID = 101
CLIENT_ID = 201
APARTMENT_ID = 301
BUILDING_ID = 401


def backend_response(data: dict) -> BackendResponse:
    return BackendResponse(
        request_id="backend_req_dialog",
        success=True,
        data=data,
        error=None,
    )


def deal_data() -> dict:
    return {
        "id": DEAL_ID,
        "client_id": CLIENT_ID,
        "apartment_id": APARTMENT_ID,
    }


def analysis_result() -> DialogAnalyzeResult:
    return DialogAnalyzeResult(
        deal_id=DEAL_ID,
        client_id=CLIENT_ID,
        analysis=ClientFacts(
            budget_min=None,
            budget_max=15_000_000,
            rooms=2,
            floor_min=None,
            floor_max=None,
            parking_required=None,
            renovation_required=None,
            preferred_district=None,
            purchase_timeline=None,
            important_factors=[],
            objections=[],
            summary="Клиент ищет двухкомнатную квартиру.",
        ),
        preferences_updated=True,
    )


def make_message(body: dict, routing_key: str) -> Mock:
    message = Mock()
    message.body = json.dumps(body).encode()
    message.routing_key = routing_key
    message.exchange = "app.topic"
    message.reply_to = "rpc.reply"
    message.correlation_id = "correlation-1"
    message.headers = {}
    message.ack = AsyncMock()
    message.nack = AsyncMock()
    message.reject = AsyncMock()
    return message


def make_consumer(dialog_service: Mock, publisher: Mock) -> tuple[AgentConsumer, Mock]:
    router = Mock(route=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        Mock(generate=AsyncMock()),
        router,
        Mock(generate=AsyncMock()),
        Mock(generate=AsyncMock()),
        dialog_service=dialog_service,
    )
    return consumer, router


def test_dialog_request_schemas_and_selected_text_normalization() -> None:
    analyze = DialogAnalyzeRequest.model_validate(
        {
            "request_id": REQUEST_ID,
            "action": "dialog.analyze",
            "payload": {"deal_id": DEAL_ID, "user_id": USER_ID},
        }
    )
    selected = DialogReplyAssistRequest.model_validate(
        {
            "request_id": REQUEST_ID,
            "action": "dialog.reply_assist",
            "payload": {
                "deal_id": DEAL_ID,
                "user_id": USER_ID,
                "selected_text": "  У конкурента дешевле  ",
            },
        }
    )
    blank = DialogReplyAssistRequest.model_validate(
        {
            "request_id": REQUEST_ID,
            "action": "dialog.reply_assist",
            "payload": {
                "deal_id": DEAL_ID,
                "user_id": USER_ID,
                "selected_text": "   ",
            },
        }
    )

    assert analyze.payload.deal_id == DEAL_ID
    assert selected.payload.selected_text == "У конкурента дешевле"
    assert blank.payload.selected_text is None


def test_dialog_request_rejects_wrong_action_and_extra_fields() -> None:
    with pytest.raises(ValidationError):
        DialogAnalyzeRequest.model_validate(
            {
                "request_id": REQUEST_ID,
                "action": "chat",
                "payload": {
                    "deal_id": DEAL_ID,
                    "user_id": USER_ID,
                    "message": "лишнее поле",
                },
            }
        )


@pytest.mark.asyncio
async def test_dialog_analyze_returns_structured_result() -> None:
    analytics = Mock(analyze_dialog=AsyncMock(return_value=analysis_result()))
    service = DialogService(analytics, Mock(), Mock(), Mock())

    result = await service.analyze(DEAL_ID)

    analytics.analyze_dialog.assert_awaited_once_with(DEAL_ID)
    assert result.analysis.rooms == 2
    assert result.preferences_updated is True


@pytest.mark.asyncio
async def test_dialog_analyze_uses_only_client_messages_and_saves_preferences() -> None:
    facts = ClientFacts(
        budget_min=None,
        budget_max=15_000_000,
        rooms=2,
        floor_min=7,
        floor_max=None,
        parking_required=True,
        renovation_required=None,
        preferred_district=None,
        purchase_timeline=None,
        important_factors=[],
        objections=["У конкурента дешевле"],
        summary="Клиент ищет двухкомнатную квартиру.",
    )
    llm = Mock(parse_structured=AsyncMock(return_value=facts))
    deal_tools = Mock(
        get_deal=AsyncMock(return_value=backend_response(deal_data()))
    )
    messages_tools = Mock(
        get_deal_messages=AsyncMock(
            return_value=backend_response(
                {
                    "messages": [
                        {
                            "id": 601,
                            "direction": "manager_to_client",
                            "body": "Менеджер предлагает три комнаты.",
                        },
                        {
                            "id": 602,
                            "direction": "client_to_manager",
                            "body": "Мне нужны две комнаты до 15 миллионов.",
                        },
                    ]
                }
            )
        )
    )
    client_tools = Mock(update_client_preferences=AsyncMock())
    analytics = AnalyticsAgent(
        llm,
        deal_tools,
        Mock(),
        Mock(),
        Mock(),
        messages_tools,
        client_tools,
    )

    service = DialogService(analytics, Mock(), deal_tools, messages_tools)
    result = await service.analyze(DEAL_ID)

    prompt = llm.parse_structured.await_args.args[0]
    assert "Мне нужны две комнаты" in prompt
    assert "Менеджер предлагает три комнаты" not in prompt
    client_tools.update_client_preferences.assert_awaited_once_with(
        CLIENT_ID,
        preferences={
            "rooms": 2,
            "floor_min": 7,
            "parking": True,
            "objections": ["У конкурента дешевле"],
        },
        budget_max=15_000_000,
    )
    assert result.analysis == facts
    assert result.analysis.budget_max == 15_000_000
    assert result.analysis.rooms == 2
    assert result.analysis.floor_min == 7
    assert result.analysis.parking_required is True
    assert result.analysis.objections == ["У конкурента дешевле"]
    assert result.preferences_updated is True


@pytest.mark.asyncio
async def test_reply_assist_prefers_selected_text_and_does_not_load_history() -> None:
    deal_tools = Mock(
        get_deal=AsyncMock(return_value=backend_response(deal_data()))
    )
    messages_tools = Mock(get_deal_messages=AsyncMock())
    negotiation = Mock(
        reply_assist=AsyncMock(
            return_value=(
                ReplyAssistAnalysis(
                    intent="price_objection",
                    summary="Клиент возражает против цены.",
                ),
                "Рекомендуемый ответ",
            )
        )
    )
    service = DialogService(Mock(), negotiation, deal_tools, messages_tools)

    result = await service.reply_assist(DEAL_ID, "У конкурента дешевле")

    messages_tools.get_deal_messages.assert_not_awaited()
    target, deal = negotiation.reply_assist.await_args.args
    assert target == "У конкурента дешевле"
    assert deal.id == DEAL_ID
    assert result.source == "selected_text"
    assert result.suggested_reply == "Рекомендуемый ответ"


@pytest.mark.asyncio
async def test_reply_assist_never_runs_analytics_preference_update() -> None:
    analytics = Mock(analyze_dialog=AsyncMock())
    service = DialogService(
        analytics,
        Mock(
            reply_assist=AsyncMock(
                return_value=(
                    ReplyAssistAnalysis(intent="other_objection", summary="Возражение"),
                    "Ответ",
                )
            )
        ),
        Mock(get_deal=AsyncMock(return_value=backend_response(deal_data()))),
        Mock(),
    )

    await service.reply_assist(DEAL_ID, "Мне это не подходит")

    analytics.analyze_dialog.assert_not_awaited()


@pytest.mark.asyncio
async def test_reply_assist_selects_last_client_message_not_manager_message() -> None:
    messages = [
        {
            "id": 603,
            "direction": "client_to_manager",
            "body": "Первое сообщение клиента",
        },
        {
            "id": 604,
            "direction": "client_to_manager",
            "body": "Последнее сообщение клиента",
        },
        {
            "id": 605,
            "direction": "manager_to_client",
            "body": "Последнее сообщение менеджера",
        },
    ]
    deal_tools = Mock(
        get_deal=AsyncMock(return_value=backend_response(deal_data()))
    )
    messages_tools = Mock(
        get_deal_messages=AsyncMock(
            return_value=backend_response({"messages": messages})
        )
    )
    negotiation = Mock(
        reply_assist=AsyncMock(
            return_value=(
                ReplyAssistAnalysis(intent="general_question", summary="Вопрос"),
                "Ответ",
            )
        )
    )
    service = DialogService(Mock(), negotiation, deal_tools, messages_tools)

    result = await service.reply_assist(DEAL_ID, None)

    messages_tools.get_deal_messages.assert_awaited_once_with(DEAL_ID, limit=30)
    assert negotiation.reply_assist.await_args.args[0] == "Последнее сообщение клиента"
    assert result.source == "last_client_message"


@pytest.mark.asyncio
async def test_reply_assist_empty_history_is_controlled() -> None:
    deal_tools = Mock(
        get_deal=AsyncMock(return_value=backend_response(deal_data()))
    )
    messages_tools = Mock(
        get_deal_messages=AsyncMock(
            return_value=backend_response({"messages": []})
        )
    )
    negotiation = Mock(reply_assist=AsyncMock())
    service = DialogService(Mock(), negotiation, deal_tools, messages_tools)

    with pytest.raises(DialogCommandError) as error:
        await service.reply_assist(DEAL_ID, None)

    assert error.value.code == "NO_CLIENT_MESSAGES"
    negotiation.reply_assist.assert_not_awaited()


@pytest.mark.asyncio
async def test_reply_assist_backend_timeout_propagates_for_retry_policy() -> None:
    timeout = BackendRpcTimeoutError("timeout", code="TIMEOUT")
    service = DialogService(
        Mock(),
        Mock(),
        Mock(get_deal=AsyncMock(side_effect=timeout)),
        Mock(),
    )

    with pytest.raises(BackendRpcTimeoutError):
        await service.reply_assist(DEAL_ID, "Текст")


@pytest.mark.asyncio
async def test_negotiation_reply_assist_uses_factual_context() -> None:
    llm = Mock(
        parse_structured=AsyncMock(
            return_value=ReplyAssistAnalysis(
                intent="competitor_comparison",
                summary="Клиент сравнивает предложение с конкурентом.",
            )
        ),
        generate=AsyncMock(return_value="Фактический ответ"),
    )
    apartment_tools = Mock(
        get_apartment=AsyncMock(
            return_value=backend_response(
                {
                    "id": APARTMENT_ID,
                    "building_id": BUILDING_ID,
                    "price": 14_200_000,
                }
            )
        )
    )
    building_tools = Mock(
        get_building=AsyncMock(
            return_value=backend_response(
                {
                    "id": BUILDING_ID,
                    "district": "Центральный",
                    "planned_delivery": "2027-06-01",
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
                            "price_per_sqm": 210_000,
                        }
                    ]
                }
            )
        )
    )
    agent = NegotiationAgent(
        llm,
        Mock(),
        apartment_tools,
        building_tools,
        competitor_tools,
    )
    deal = DealFacts.model_validate(deal_data())

    analysis, reply = await agent.reply_assist("У конкурента дешевле", deal)

    assert analysis.intent == "competitor_comparison"
    assert reply == "Фактический ответ"
    prompt = llm.generate.await_args.args[0]
    assert "ЖК Конкурент" in prompt
    assert "14200000" in prompt
    assert "У конкурента дешевле" in prompt


@pytest.mark.asyncio
async def test_reply_assist_retries_unsupported_layout_claim() -> None:
    llm = Mock(
        parse_structured=AsyncMock(
            return_value=ReplyAssistAnalysis(
                intent="competitor_comparison",
                summary="Клиент сравнивает цену с конкурентом.",
            )
        ),
        generate=AsyncMock(
            side_effect=[
                "У нас больше вариантов отделки и разнообразие планировок.",
                "У нас больше вариантов отделки.",
            ]
        ),
    )
    apartment_tools = Mock(
        get_apartment=AsyncMock(
            return_value=backend_response(
                {"id": APARTMENT_ID, "building_id": BUILDING_ID}
            )
        )
    )
    building_tools = Mock(
        get_building=AsyncMock(
            return_value=backend_response(
                {"id": BUILDING_ID, "district": "Центральный"}
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
                            "disadvantages": "Меньше вариантов отделки",
                        }
                    ]
                }
            )
        )
    )
    agent = NegotiationAgent(
        llm,
        Mock(),
        apartment_tools,
        building_tools,
        competitor_tools,
    )

    _, reply = await agent.reply_assist(
        "У конкурента дешевле",
        DealFacts.model_validate(deal_data()),
    )

    assert "вариантов отделки" in reply
    assert "планиров" not in reply.casefold()
    assert llm.generate.await_count == 2
    first_prompt = llm.generate.await_args_list[0].args[0]
    assert "Разрешённые факты" in first_prompt
    assert "Меньше вариантов отделки" in first_prompt


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "unsupported_reply",
    [
        "Рядом есть метро и удобный транспорт.",
        "Доступна выгодная ипотека и большая скидка.",
        "Рядом развитая инфраструктура, школы и магазины.",
    ],
)
async def test_reply_assist_rejects_other_unconfirmed_advantages(
    unsupported_reply: str,
) -> None:
    llm = Mock(
        parse_structured=AsyncMock(
            return_value=ReplyAssistAnalysis(
                intent="price_objection",
                summary="Клиент считает цену высокой.",
            )
        ),
        generate=AsyncMock(
            side_effect=[unsupported_reply, "Для сравнения недостаточно данных."],
        ),
    )
    agent = NegotiationAgent(
        llm,
        Mock(),
        Mock(
            get_apartment=AsyncMock(
                return_value=backend_response(
                    {"id": APARTMENT_ID, "building_id": BUILDING_ID}
                )
            )
        ),
        Mock(
            get_building=AsyncMock(
                return_value=backend_response(
                    {"id": BUILDING_ID, "district": "Центральный"}
                )
            )
        ),
        Mock(
            list_competitors=AsyncMock(
                return_value=backend_response({"competitors": []})
            )
        ),
    )

    _, reply = await agent.reply_assist(
        "Клиент считает цену высокой",
        DealFacts.model_validate(deal_data()),
    )

    assert reply == "Для сравнения недостаточно данных."
    assert llm.generate.await_count == 2


@pytest.mark.asyncio
async def test_invalid_reply_analysis_has_safe_fallback() -> None:
    llm = Mock(
        parse_structured=AsyncMock(side_effect=[ValueError("bad"), ValueError("bad")]),
        generate=AsyncMock(return_value="Ответ"),
    )
    agent = NegotiationAgent(
        llm,
        Mock(),
        Mock(
            get_apartment=AsyncMock(
                return_value=backend_response(
                    {"id": APARTMENT_ID, "building_id": BUILDING_ID}
                )
            )
        ),
        Mock(
            get_building=AsyncMock(
                return_value=backend_response(
                    {"id": BUILDING_ID, "district": "Центральный"}
                )
            )
        ),
        Mock(
            list_competitors=AsyncMock(
                return_value=backend_response({"competitors": []})
            )
        ),
    )

    analysis, _ = await agent.reply_assist(
        "Непонятное сообщение",
        DealFacts.model_validate(deal_data()),
    )

    assert analysis.intent == "unknown"
    assert llm.parse_structured.await_count == 2


@pytest.mark.asyncio
async def test_dialog_analyze_consumer_bypasses_router_and_publishes_structure() -> None:
    dialog_service = Mock(
        analyze=AsyncMock(return_value=analysis_result()),
        reply_assist=AsyncMock(),
    )
    publisher = Mock(publish=AsyncMock())
    consumer, router = make_consumer(dialog_service, publisher)
    body = {
        "request_id": REQUEST_ID,
        "action": "dialog.analyze",
        "payload": {"deal_id": DEAL_ID, "user_id": USER_ID},
    }
    message = make_message(body, "agent.dialog.analyze")

    await consumer.handle_message(message)

    dialog_service.analyze.assert_awaited_once_with(DEAL_ID)
    router.route.assert_not_awaited()
    response = publisher.publish.await_args.kwargs["response"]
    assert response.success is True
    assert response.data.analysis.budget_max == 15_000_000
    message.ack.assert_awaited_once()


@pytest.mark.asyncio
async def test_reply_assist_consumer_bypasses_router() -> None:
    result = DialogReplyAssistResult(
        deal_id=DEAL_ID,
        source="selected_text",
        analysis=ReplyAssistAnalysis(
            intent="price_objection",
            summary="Возражение по цене",
        ),
        suggested_reply="Готовый ответ",
    )
    dialog_service = Mock(
        analyze=AsyncMock(),
        reply_assist=AsyncMock(return_value=result),
    )
    publisher = Mock(publish=AsyncMock())
    consumer, router = make_consumer(dialog_service, publisher)
    body = {
        "request_id": REQUEST_ID,
        "action": "dialog.reply_assist",
        "payload": {
            "deal_id": DEAL_ID,
            "user_id": USER_ID,
            "selected_text": "  Дорого  ",
        },
    }
    message = make_message(body, "agent.dialog.reply_assist")

    await consumer.handle_message(message)

    dialog_service.reply_assist.assert_awaited_once_with(DEAL_ID, "Дорого")
    router.route.assert_not_awaited()
    response = publisher.publish.await_args.kwargs["response"]
    assert response.data.suggested_reply == "Готовый ответ"


@pytest.mark.asyncio
async def test_dialog_invalid_request_returns_validation_error() -> None:
    publisher = Mock(publish=AsyncMock())
    consumer, _ = make_consumer(Mock(), publisher)
    body = {
        "request_id": REQUEST_ID,
        "action": "dialog.analyze",
        "payload": {"deal_id": "invalid", "user_id": USER_ID},
    }

    await consumer.handle_message(make_message(body, "agent.dialog.analyze"))

    response = publisher.publish.await_args.kwargs["response"]
    assert response.success is False
    assert response.error.code == "VALIDATION_ERROR"


@pytest.mark.asyncio
async def test_dialog_backend_timeout_uses_existing_retry_policy() -> None:
    dialog_service = Mock(
        analyze=AsyncMock(side_effect=BackendRpcTimeoutError("timeout", code="TIMEOUT"))
    )
    publisher = Mock(publish=AsyncMock(), retry=AsyncMock())
    consumer, _ = make_consumer(dialog_service, publisher)
    body = {
        "request_id": REQUEST_ID,
        "action": "dialog.analyze",
        "payload": {"deal_id": DEAL_ID, "user_id": USER_ID},
    }
    message = make_message(body, "agent.dialog.analyze")

    await consumer.handle_message(message)

    publisher.retry.assert_awaited_once_with(message, 1)
    publisher.publish.assert_not_awaited()


@pytest.mark.asyncio
async def test_dialog_gigachat_error_uses_existing_retry_policy() -> None:
    dialog_service = Mock(
        reply_assist=AsyncMock(side_effect=httpx.ConnectError("network"))
    )
    publisher = Mock(publish=AsyncMock(), retry=AsyncMock())
    consumer, _ = make_consumer(dialog_service, publisher)
    body = {
        "request_id": REQUEST_ID,
        "action": "dialog.reply_assist",
        "payload": {"deal_id": DEAL_ID, "user_id": USER_ID},
    }
    message = make_message(body, "agent.dialog.reply_assist")

    await consumer.handle_message(message)

    publisher.retry.assert_awaited_once_with(message, 1)
    publisher.publish.assert_not_awaited()


@pytest.mark.asyncio
async def test_dialog_controlled_errors_are_returned_without_hallucinated_data() -> None:
    for error, expected_code in (
        (NoClientMessagesError("empty"), "NO_CLIENT_MESSAGES"),
        (ClientFactsExtractionError("invalid"), "ANALYSIS_ERROR"),
        (
            DialogCommandError("NO_CLIENT_MESSAGES", "Нет сообщений клиента"),
            "NO_CLIENT_MESSAGES",
        ),
        (BackendRpcError("not found", code="DEAL_NOT_FOUND"), "DEAL_NOT_FOUND"),
    ):
        dialog_service = Mock(analyze=AsyncMock(side_effect=error))
        publisher = Mock(publish=AsyncMock())
        consumer, _ = make_consumer(dialog_service, publisher)
        body = {
            "request_id": REQUEST_ID,
            "action": "dialog.analyze",
            "payload": {"deal_id": DEAL_ID, "user_id": USER_ID},
        }

        await consumer.handle_message(make_message(body, "agent.dialog.analyze"))

        response = publisher.publish.await_args.kwargs["response"]
        assert response.success is False
        assert response.data is None
        assert response.error.code == expected_code


@pytest.mark.asyncio
async def test_existing_queue_is_bound_to_both_dialog_routing_keys(monkeypatch) -> None:
    exchange = Mock()
    queue = Mock(bind=AsyncMock())
    channel = Mock(
        set_qos=AsyncMock(),
        declare_exchange=AsyncMock(return_value=exchange),
        declare_queue=AsyncMock(return_value=queue),
    )
    connection = Mock(channel=AsyncMock(return_value=channel), is_closed=False)
    connect = AsyncMock(return_value=connection)
    monkeypatch.setattr("app.broker.connection.aio_pika.connect_robust", connect)
    settings = Mock(
        rabbitmq_url=Mock(get_secret_value=Mock(return_value="amqp://guest:guest@localhost/")),
        rabbitmq_prefetch_count=10,
        rabbitmq_exchange="app.topic",
        rabbitmq_queue="agent-service",
        rabbitmq_routing_key="agent.chat.request",
    )

    broker = RabbitMQConnection(settings)
    await broker.connect()

    bound_keys = {call.kwargs["routing_key"] for call in queue.bind.await_args_list}
    assert bound_keys == {"agent.chat.request", *DIALOG_ROUTING_KEYS}
