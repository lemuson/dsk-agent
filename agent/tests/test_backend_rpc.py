import asyncio
import json
from unittest.mock import AsyncMock, Mock
import pytest

from app.broker.backend_rpc import (
    BackendResponseValidationError,
    BackendRpcClient,
    BackendRpcError,
    BackendRpcTimeoutError,
)


async def make_client() -> tuple[BackendRpcClient, Mock, Mock, Mock]:
    reply_queue = Mock()
    reply_queue.name = "amq.gen-callback"
    reply_queue.consume = AsyncMock(return_value="consumer-tag")
    reply_queue.cancel = AsyncMock()
    channel = Mock(declare_queue=AsyncMock(return_value=reply_queue))
    exchange = Mock(publish=AsyncMock())
    client = BackendRpcClient(channel, exchange)
    await client.start()
    return client, channel, exchange, reply_queue


def make_response_message(correlation_id: str, body: object) -> Mock:
    message = Mock()
    message.correlation_id = correlation_id
    if isinstance(body, bytes):
        message.body = body
    else:
        message.body = json.dumps(body).encode("utf-8")
    message.ack = AsyncMock()
    return message


@pytest.mark.asyncio
async def test_successful_call_has_valid_rpc_properties_and_cleans_pending() -> None:
    client, channel, exchange, reply_queue = await make_client()
    published: list[tuple[object, str]] = []

    async def publish(message: object, routing_key: str) -> None:
        published.append((message, routing_key))
        request = json.loads(message.body)
        response = make_response_message(
            message.correlation_id,
            {
                "request_id": request["request_id"],
                "success": True,
                "data": {"id": "deal-1"},
                "error": None,
            },
        )
        await client._handle_response(response)

    exchange.publish.side_effect = publish

    response = await client.call(
        routing_key="backend.deal.get",
        action="deal.get",
        payload={"deal_id": 101},
    )

    channel.declare_queue.assert_awaited_once_with(
        name="",
        exclusive=True,
        auto_delete=True,
    )
    reply_queue.consume.assert_awaited_once()
    request_message, routing_key = published[0]
    request = json.loads(request_message.body)
    assert isinstance(request["request_id"], str) and request["request_id"]
    assert isinstance(request_message.correlation_id, str)
    assert request_message.correlation_id
    assert request_message.reply_to == "amq.gen-callback"
    assert request_message.content_type == "application/json"
    assert routing_key == "backend.deal.get"
    assert response.data == {"id": "deal-1"}
    assert client.pending_count == 0
    await client.close()


@pytest.mark.asyncio
async def test_parallel_calls_are_matched_by_correlation_id() -> None:
    client, _, exchange, _ = await make_client()
    published: list[object] = []
    both_published = asyncio.Event()

    async def capture(message: object, routing_key: str) -> None:
        published.append(message)
        if len(published) == 2:
            both_published.set()

    exchange.publish.side_effect = capture
    first_task = asyncio.create_task(
        client.call("backend.deal.get", "deal.get", {"deal_id": 101})
    )
    second_task = asyncio.create_task(
        client.call("backend.building.get", "building.get", {"building_id": 401})
    )
    await asyncio.wait_for(both_published.wait(), timeout=1)

    first_request = json.loads(published[0].body)
    second_request = json.loads(published[1].body)
    await client._handle_response(
        make_response_message(
            published[1].correlation_id,
            {
                "request_id": second_request["request_id"],
                "success": True,
                "data": {"result": "second"},
                "error": None,
            },
        )
    )
    await client._handle_response(
        make_response_message(
            published[0].correlation_id,
            {
                "request_id": first_request["request_id"],
                "success": True,
                "data": {"result": "first"},
                "error": None,
            },
        )
    )

    first, second = await asyncio.gather(first_task, second_task)
    assert first.data == {"result": "first"}
    assert second.data == {"result": "second"}
    assert client.pending_count == 0
    await client.close()


@pytest.mark.asyncio
async def test_timeout_removes_pending_future() -> None:
    client, _, _, _ = await make_client()

    with pytest.raises(BackendRpcTimeoutError) as captured:
        await client.call("backend.deal.get", "deal.get", {}, timeout=0.01)

    assert captured.value.code == "TIMEOUT"
    assert client.pending_count == 0
    await client.close()


@pytest.mark.asyncio
async def test_backend_error_response_raises_backend_specific_error() -> None:
    client, _, exchange, _ = await make_client()

    async def publish(message: object, routing_key: str) -> None:
        request = json.loads(message.body)
        await client._handle_response(
            make_response_message(
                message.correlation_id,
                {
                    "request_id": request["request_id"],
                    "success": False,
                    "data": None,
                    "error": {"code": "DEAL_NOT_FOUND", "message": "Deal not found"},
                },
            )
        )

    exchange.publish.side_effect = publish

    with pytest.raises(BackendRpcError) as captured:
        await client.call("backend.deal.get", "deal.get", {})

    assert captured.value.code == "DEAL_NOT_FOUND"
    assert captured.value.response is not None
    assert client.pending_count == 0
    await client.close()


@pytest.mark.asyncio
async def test_invalid_json_response_raises_validation_error() -> None:
    client, _, exchange, _ = await make_client()

    async def publish(message: object, routing_key: str) -> None:
        await client._handle_response(
            make_response_message(message.correlation_id, b"{not-json")
        )

    exchange.publish.side_effect = publish

    with pytest.raises(BackendResponseValidationError) as captured:
        await client.call("backend.deal.get", "deal.get", {})

    assert captured.value.code == "INVALID_RESPONSE_JSON"
    assert client.pending_count == 0
    await client.close()


@pytest.mark.asyncio
async def test_invalid_response_schema_raises_validation_error() -> None:
    client, _, exchange, _ = await make_client()

    async def publish(message: object, routing_key: str) -> None:
        request = json.loads(message.body)
        await client._handle_response(
            make_response_message(
                message.correlation_id,
                {
                    "request_id": request["request_id"],
                    "success": True,
                    "data": None,
                    "error": None,
                },
            )
        )

    exchange.publish.side_effect = publish

    with pytest.raises(BackendResponseValidationError) as captured:
        await client.call("backend.deal.get", "deal.get", {})

    assert captured.value.code == "INVALID_RESPONSE_SCHEMA"
    assert client.pending_count == 0
    await client.close()


@pytest.mark.asyncio
async def test_close_completes_and_removes_pending_requests() -> None:
    client, _, exchange, reply_queue = await make_client()
    request_published = asyncio.Event()

    async def publish(message: object, routing_key: str) -> None:
        request_published.set()

    exchange.publish.side_effect = publish
    call_task = asyncio.create_task(
        client.call("backend.deal.get", "deal.get", {"deal_id": 101})
    )
    await asyncio.wait_for(request_published.wait(), timeout=1)

    await client.close()

    with pytest.raises(BackendRpcError) as captured:
        await call_task
    assert captured.value.code == "CLIENT_CLOSED"
    assert client.pending_count == 0
    reply_queue.cancel.assert_awaited_once_with("consumer-tag")
