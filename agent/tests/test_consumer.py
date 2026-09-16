import json
from unittest.mock import AsyncMock, Mock

import httpx
import pytest
from gigachat.exceptions import RateLimitError, ServerError

from app.agents.router import RouteDecision
from app.broker.backend_rpc import BackendRpcError
from app.broker.consumer import AgentConsumer
from app.broker.publisher import RpcPublisher


REQUEST = {
    "request_id": "req_test_123",
    "action": "chat",
    "payload": {
        "user_id": 15,
        "session_id": 42,
        "deal_id": None,
        "message": "Привет",
    },
}

def make_router(agent: str = "general", intent: str = "general_chat") -> Mock:
    decision = RouteDecision.model_validate(
        {"agent": agent, "intent": intent, "confidence": 0.9}
    )
    return Mock(route=AsyncMock(return_value=decision))


def make_negotiation_agent(answer: str = "Ответ на возражение") -> Mock:
    return Mock(generate=AsyncMock(return_value=answer))


def make_analytics_agent(answer: str = "Аналитический ответ") -> Mock:
    return Mock(generate=AsyncMock(return_value=answer))


def make_offer_agent(answer: str = "Коммерческое предложение") -> Mock:
    return Mock(
        generate=AsyncMock(return_value=answer),
        resume_pending=AsyncMock(return_value=None),
    )


def make_message(
    body: dict = REQUEST,
    reply_to: str | None = "rpc.reply",
    correlation_id: str | None = "correlation-1",
    retry_count: int = 0,
) -> Mock:
    message = Mock()
    message.body = json.dumps(body).encode()
    message.reply_to = reply_to
    message.correlation_id = correlation_id
    message.headers = {"x-retry-count": retry_count} if retry_count else {}
    message.ack = AsyncMock()
    message.nack = AsyncMock()
    message.reject = AsyncMock()
    return message


@pytest.mark.asyncio
async def test_success_is_published_before_ack() -> None:
    llm = Mock(generate=AsyncMock(return_value="Здравствуйте"))
    publisher = Mock(publish=AsyncMock())
    router = make_router()
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        router,
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message()

    await consumer.handle_message(message)

    llm.generate.assert_awaited_once_with("Привет")
    router.route.assert_awaited_once_with("Привет")
    publisher.publish.assert_awaited_once()
    published = publisher.publish.await_args.kwargs["response"]
    assert published.success is True
    assert published.data.message == "Здравствуйте"
    assert published.data.agent == "general"
    assert published.data.intent == "general_chat"
    assert published.request_id == REQUEST["request_id"]
    assert publisher.publish.await_args.kwargs["correlation_id"] == "correlation-1"
    message.ack.assert_awaited_once()
    message.nack.assert_not_awaited()


@pytest.mark.asyncio
async def test_llm_error_returns_rpc_error_and_acks() -> None:
    llm = Mock(generate=AsyncMock(side_effect=RuntimeError("API unavailable")))
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message()

    await consumer.handle_message(message)

    published = publisher.publish.await_args.kwargs["response"]
    assert published.success is False
    assert published.error.code == "INTERNAL_ERROR"
    message.ack.assert_awaited_once()


@pytest.mark.asyncio
async def test_validation_error_uses_contract_error_code() -> None:
    invalid_request = {**REQUEST, "action": "unknown"}
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        Mock(),
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message(body=invalid_request)

    await consumer.handle_message(message)

    published = publisher.publish.await_args.kwargs["response"]
    assert published.success is False
    assert published.error.code == "VALIDATION_ERROR"
    message.ack.assert_awaited_once()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    "temporary_error",
    [
        httpx.ConnectError("network error"),
        httpx.TimeoutException("timeout"),
        RateLimitError("https://example.test", 429, b"rate limit", None),
        ServerError("https://example.test", 503, b"unavailable", None),
    ],
)
async def test_temporary_llm_error_is_republished_with_retry_count(
    temporary_error: Exception,
) -> None:
    llm = Mock(generate=AsyncMock(side_effect=temporary_error))
    publisher = Mock(publish=AsyncMock(), retry=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message()

    await consumer.handle_message(message)

    publisher.retry.assert_awaited_once_with(message, 1)
    publisher.publish.assert_not_awaited()
    message.ack.assert_awaited_once()
    message.nack.assert_not_awaited()


@pytest.mark.asyncio
async def test_failed_retry_republish_falls_back_to_nack() -> None:
    llm = Mock(generate=AsyncMock(side_effect=httpx.ConnectError("network error")))
    publisher = Mock(
        publish=AsyncMock(),
        retry=AsyncMock(side_effect=RuntimeError("publish error")),
    )
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message()

    await consumer.handle_message(message)

    message.nack.assert_awaited_once_with(requeue=True)
    message.ack.assert_not_awaited()


@pytest.mark.asyncio
async def test_temporary_llm_error_after_retry_limit_returns_error() -> None:
    llm = Mock(generate=AsyncMock(side_effect=httpx.TimeoutException("timeout")))
    publisher = Mock(publish=AsyncMock(), retry=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message(retry_count=2)

    await consumer.handle_message(message)

    publisher.retry.assert_not_awaited()
    published = publisher.publish.await_args.kwargs["response"]
    assert published.success is False
    assert published.error.code == "INTERNAL_ERROR"
    message.ack.assert_awaited_once()


@pytest.mark.asyncio
async def test_publish_error_requeues_without_ack() -> None:
    llm = Mock(generate=AsyncMock(return_value="Ответ"))
    publisher = Mock(publish=AsyncMock(side_effect=RuntimeError("broker error")))
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message()

    await consumer.handle_message(message)

    message.nack.assert_awaited_once_with(requeue=True)
    message.ack.assert_not_awaited()


@pytest.mark.asyncio
async def test_missing_reply_to_is_rejected() -> None:
    consumer = AgentConsumer(
        Mock(),
        Mock(),
        Mock(),
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message(reply_to=None)

    await consumer.handle_message(message)

    message.reject.assert_awaited_once_with(requeue=False)
    message.ack.assert_not_awaited()


@pytest.mark.asyncio
async def test_missing_correlation_id_is_rejected() -> None:
    consumer = AgentConsumer(
        Mock(),
        Mock(),
        Mock(),
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message(correlation_id=None)

    await consumer.handle_message(message)

    message.reject.assert_awaited_once_with(requeue=False)
    message.ack.assert_not_awaited()


@pytest.mark.asyncio
async def test_blank_correlation_id_is_rejected() -> None:
    consumer = AgentConsumer(
        Mock(),
        Mock(),
        Mock(),
        make_router(),
        make_negotiation_agent(),
        make_analytics_agent(),
    )
    message = make_message(correlation_id="   ")

    await consumer.handle_message(message)

    message.reject.assert_awaited_once_with(requeue=False)
    message.ack.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize("intent", ["handle_objection", "compare_competitor"])
async def test_negotiation_route_uses_negotiation_agent(intent: str) -> None:
    llm = Mock(generate=AsyncMock(return_value="Общий ответ"))
    negotiation_agent = make_negotiation_agent("Ответ для переговоров")
    router = make_router("negotiation", intent)
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        router,
        negotiation_agent,
        make_analytics_agent(),
    )
    deal_id = 101
    request_with_deal = {
        **REQUEST,
        "payload": {**REQUEST["payload"], "deal_id": deal_id},
    }
    message = make_message(body=request_with_deal)

    await consumer.handle_message(message)

    negotiation_agent.generate.assert_awaited_once_with(
        "Привет",
        intent,
        deal_id,
    )
    llm.generate.assert_not_awaited()
    published = publisher.publish.await_args.kwargs["response"]
    assert published.data.message == "Ответ для переговоров"
    assert published.data.agent == "negotiation"
    assert published.data.intent == intent


@pytest.mark.asyncio
@pytest.mark.parametrize("intent", ["analyze_risk", "analyze_construction"])
async def test_analytics_route_uses_analytics_agent(intent: str) -> None:
    llm = Mock(generate=AsyncMock(return_value="Общий ответ"))
    negotiation_agent = make_negotiation_agent()
    analytics_agent = make_analytics_agent("Аналитический ответ")
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router("analytics", intent),
        negotiation_agent,
        analytics_agent,
    )
    deal_id = 101
    request_with_deal = {
        **REQUEST,
        "payload": {**REQUEST["payload"], "deal_id": deal_id},
    }
    message = make_message(body=request_with_deal)

    await consumer.handle_message(message)

    analytics_agent.generate.assert_awaited_once_with(
        "Привет",
        intent,
        deal_id,
    )
    negotiation_agent.generate.assert_not_awaited()
    llm.generate.assert_not_awaited()
    published = publisher.publish.await_args.kwargs["response"]
    assert published.data.message == "Аналитический ответ"
    assert published.data.agent == "analytics"
    assert published.data.intent == intent


@pytest.mark.asyncio
async def test_extract_client_facts_uses_analytics_agent() -> None:
    llm = Mock(generate=AsyncMock(return_value="Общий ответ"))
    negotiation_agent = make_negotiation_agent()
    analytics_agent = make_analytics_agent()
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router("analytics", "extract_client_facts"),
        negotiation_agent,
        analytics_agent,
    )

    await consumer.handle_message(make_message())

    analytics_agent.generate.assert_awaited_once_with(
        "Привет",
        "extract_client_facts",
        None,
    )
    llm.generate.assert_not_awaited()
    negotiation_agent.generate.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("agent", "intent"),
    [("general", "general_chat")],
)
async def test_offer_and_general_routes_use_general_generator(
    agent: str,
    intent: str,
) -> None:
    llm = Mock(generate=AsyncMock(return_value="Общий ответ"))
    negotiation_agent = make_negotiation_agent()
    analytics_agent = make_analytics_agent()
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router(agent, intent),
        negotiation_agent,
        analytics_agent,
    )
    message = make_message()

    await consumer.handle_message(message)

    llm.generate.assert_awaited_once_with("Привет")
    negotiation_agent.generate.assert_not_awaited()
    analytics_agent.generate.assert_not_awaited()
    published = publisher.publish.await_args.kwargs["response"]
    assert published.data.agent == agent
    assert published.data.intent == intent


@pytest.mark.asyncio
@pytest.mark.parametrize("intent", ["create_offer", "calculate_offer"])
async def test_offer_routes_use_offer_agent(intent: str) -> None:
    llm = Mock(generate=AsyncMock(return_value="Общий ответ"))
    negotiation_agent = make_negotiation_agent()
    analytics_agent = make_analytics_agent()
    offer_agent = make_offer_agent()
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router("offer", intent),
        negotiation_agent,
        analytics_agent,
        offer_agent,
    )
    deal_id = 101
    request_with_deal = {
        **REQUEST,
        "payload": {**REQUEST["payload"], "deal_id": deal_id},
    }

    await consumer.handle_message(make_message(body=request_with_deal))

    offer_agent.generate.assert_awaited_once_with(
        "Привет",
        intent,
        deal_id,
        REQUEST["payload"]["user_id"],
        REQUEST["payload"]["session_id"],
    )
    negotiation_agent.generate.assert_not_awaited()
    analytics_agent.generate.assert_not_awaited()
    llm.generate.assert_not_awaited()
    published = publisher.publish.await_args.kwargs["response"]
    assert published.data.agent == "offer"
    assert published.data.intent == intent


@pytest.mark.asyncio
async def test_offer_forbidden_is_returned_as_rpc_error() -> None:
    offer_agent = make_offer_agent()
    offer_agent.generate.side_effect = BackendRpcError("forbidden", code="FORBIDDEN")
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        Mock(generate=AsyncMock()),
        make_router("offer", "create_offer"),
        make_negotiation_agent(),
        make_analytics_agent(),
        offer_agent,
    )
    request_with_deal = {
        **REQUEST,
        "payload": {**REQUEST["payload"], "deal_id": 101},
    }

    await consumer.handle_message(make_message(body=request_with_deal))

    response = publisher.publish.await_args.kwargs["response"]
    assert response.success is False
    assert response.error.code == "FORBIDDEN"


@pytest.mark.asyncio
async def test_pending_offer_bypasses_router_and_continues_in_same_session() -> None:
    llm = Mock(generate=AsyncMock(return_value="Общий ответ"))
    router = make_router("general", "unknown")
    offer_agent = make_offer_agent()
    offer_agent.resume_pending.return_value = (
        "Предложение со скидкой подготовлено",
        "create_offer",
    )
    publisher = Mock(publish=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        router,
        make_negotiation_agent(),
        make_analytics_agent(),
        offer_agent,
    )
    follow_up = {
        **REQUEST,
        "payload": {
            **REQUEST["payload"],
            "deal_id": None,
            "message": "7%",
        },
    }

    await consumer.handle_message(make_message(body=follow_up))

    offer_agent.resume_pending.assert_awaited_once_with(
        "7%",
        REQUEST["payload"]["user_id"],
        REQUEST["payload"]["session_id"],
    )
    router.route.assert_not_awaited()
    offer_agent.generate.assert_not_awaited()
    published = publisher.publish.await_args.kwargs["response"]
    assert published.data.message == "Предложение со скидкой подготовлено"
    assert published.data.agent == "offer"
    assert published.data.intent == "create_offer"


@pytest.mark.asyncio
async def test_temporary_negotiation_error_uses_existing_retry_policy() -> None:
    llm = Mock(generate=AsyncMock())
    negotiation_agent = Mock(
        generate=AsyncMock(side_effect=httpx.ConnectError("network error"))
    )
    publisher = Mock(publish=AsyncMock(), retry=AsyncMock())
    consumer = AgentConsumer(
        Mock(),
        publisher,
        llm,
        make_router("negotiation", "handle_objection"),
        negotiation_agent,
        make_analytics_agent(),
    )
    message = make_message()

    await consumer.handle_message(message)

    publisher.retry.assert_awaited_once_with(message, 1)
    publisher.publish.assert_not_awaited()
    message.ack.assert_awaited_once()
    message.nack.assert_not_awaited()


@pytest.mark.asyncio
async def test_retry_publisher_increments_header_and_preserves_rpc_properties() -> None:
    exchange = Mock(publish=AsyncMock())
    channel = Mock(get_exchange=AsyncMock(return_value=exchange))
    publisher = RpcPublisher(channel)
    message = make_message()
    message.exchange = "app.topic"
    message.routing_key = "agent.chat.request"
    message.content_type = "application/json"
    message.content_encoding = None
    message.delivery_mode = 1
    message.priority = None
    message.message_id = None
    message.type = None
    message.user_id = None
    message.app_id = None

    await publisher.retry(message, 1)

    channel.get_exchange.assert_awaited_once_with("app.topic")
    retried_message = exchange.publish.await_args.args[0]
    assert retried_message.headers["x-retry-count"] == 1
    assert retried_message.reply_to == "rpc.reply"
    assert retried_message.correlation_id == "correlation-1"
    exchange.publish.assert_awaited_once_with(
        retried_message,
        routing_key="agent.chat.request",
    )
