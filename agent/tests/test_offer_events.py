import json
from unittest.mock import AsyncMock, Mock

import pytest
from pydantic import ValidationError

from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.broker.event_consumer import (
    APPROVED_ROUTING_KEY,
    EVENT_QUEUE_NAME,
    REJECTED_ROUTING_KEY,
    CONSTRUCTION_DELAY_ROUTING_KEY,
    OfferEventConsumer,
)
from app.graphs.offer_continuation import OfferContinuationGraph
from app.schemas.backend import BackendResponse
from app.schemas.events import OfferApprovedEvent, OfferRejectedEvent


OFFER_ID = 501
DEAL_ID = 101


def backend_response(data: dict) -> BackendResponse:
    return BackendResponse(
        request_id="99999999-9999-9999-9999-999999999999",
        success=True,
        data=data,
        error=None,
    )


def event_body(decision: str = "approved") -> dict:
    payload = {"offer_id": OFFER_ID, "deal_id": DEAL_ID}
    if decision == "rejected":
        payload["reason"] = "Слишком высокая скидка"
    return {
        "event_id": "event_offer_approved_123",
        "event_type": f"offer.{decision}",
        "occurred_at": "2026-09-08T14:30:00Z",
        "payload": payload,
    }


def make_message(decision: str = "approved", retry_count: int = 0) -> Mock:
    message = Mock()
    message.body = json.dumps(event_body(decision)).encode()
    message.routing_key = f"event.offer.{decision}"
    message.content_type = "application/json"
    message.headers = {"x-retry-count": retry_count} if retry_count else {}
    message.ack = AsyncMock()
    message.nack = AsyncMock()
    message.reject = AsyncMock()
    return message


def make_offer_tools(status: str = "approved") -> Mock:
    return Mock(
        get_offer=AsyncMock(
            return_value=backend_response(
                {"id": OFFER_ID, "deal_id": DEAL_ID, "status": status}
            )
        ),
        generate_offer_pdf=AsyncMock(
            return_value=backend_response(
                {
                    "offer_id": OFFER_ID,
                    "document_url": "/documents/offers/123.pdf",
                }
            )
        ),
    )


def test_offer_events_pass_strict_validation() -> None:
    approved = OfferApprovedEvent.model_validate(event_body("approved"))
    rejected = OfferRejectedEvent.model_validate(event_body("rejected"))

    assert approved.payload.offer_id == OFFER_ID
    assert rejected.payload.reason == "Слишком высокая скидка"

    invalid = event_body("approved")
    invalid["payload"]["unexpected"] = True
    with pytest.raises(ValidationError):
        OfferApprovedEvent.model_validate(invalid)


@pytest.mark.asyncio
async def test_approved_event_checks_backend_status_and_generates_pdf() -> None:
    offer_tools = make_offer_tools()
    graph = OfferContinuationGraph(offer_tools)

    result = await graph.run(
        event_type="offer.approved",
        offer_id=OFFER_ID,
        deal_id=DEAL_ID,
    )

    offer_tools.get_offer.assert_awaited_once_with(OFFER_ID)
    offer_tools.generate_offer_pdf.assert_awaited_once_with(OFFER_ID)
    assert result["status"] == "DONE"
    assert result["document_url"] == "/documents/offers/123.pdf"


@pytest.mark.asyncio
async def test_rejected_event_does_not_load_offer_or_generate_pdf() -> None:
    offer_tools = make_offer_tools(status="rejected")
    graph = OfferContinuationGraph(offer_tools)

    result = await graph.run(
        event_type="offer.rejected",
        offer_id=OFFER_ID,
        deal_id=DEAL_ID,
        reason="Слишком высокая скидка",
    )

    assert result["status"] == "REJECTED"
    offer_tools.get_offer.assert_not_awaited()
    offer_tools.generate_offer_pdf.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize("status", ["pending_approval", "rejected"])
async def test_non_approved_backend_status_skips_pdf(status: str) -> None:
    offer_tools = make_offer_tools(status=status)
    graph = OfferContinuationGraph(offer_tools)

    result = await graph.run(
        event_type="offer.approved",
        offer_id=OFFER_ID,
        deal_id=DEAL_ID,
    )

    assert result["status"] == "SKIPPED"
    offer_tools.generate_offer_pdf.assert_not_awaited()


@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("error", "retryable"),
    [
        (BackendRpcTimeoutError("timeout", code="TIMEOUT"), True),
        (BackendRpcError("missing", code="OFFER_NOT_FOUND"), False),
    ],
)
async def test_get_offer_error_is_controlled(
    error: BackendRpcError,
    retryable: bool,
) -> None:
    offer_tools = make_offer_tools()
    offer_tools.get_offer.side_effect = error
    graph = OfferContinuationGraph(offer_tools)

    result = await graph.run(
        event_type="offer.approved",
        offer_id=OFFER_ID,
        deal_id=DEAL_ID,
    )

    assert result["status"] == "ERROR"
    assert result["retryable"] is retryable
    offer_tools.generate_offer_pdf.assert_not_awaited()


@pytest.mark.asyncio
async def test_generate_pdf_error_is_controlled() -> None:
    offer_tools = make_offer_tools()
    offer_tools.generate_offer_pdf.side_effect = BackendRpcError(
        "generation failed",
        code="PDF_ERROR",
    )
    graph = OfferContinuationGraph(offer_tools)

    result = await graph.run(
        event_type="offer.approved",
        offer_id=OFFER_ID,
        deal_id=DEAL_ID,
    )

    assert result["status"] == "ERROR"
    assert result["retryable"] is False


@pytest.mark.asyncio
async def test_invalid_event_is_rejected_without_requeue() -> None:
    continuation = Mock(run=AsyncMock())
    consumer = OfferEventConsumer(Mock(), Mock(), continuation)
    message = make_message()
    message.body = b"not-json"

    await consumer.handle_message(message)

    message.reject.assert_awaited_once_with(requeue=False)
    message.ack.assert_not_awaited()
    continuation.run.assert_not_awaited()


@pytest.mark.asyncio
async def test_temporary_error_is_republished_with_bounded_retry_header() -> None:
    continuation = Mock(
        run=AsyncMock(
            return_value={
                "status": "ERROR",
                "error": "TIMEOUT",
                "retryable": True,
            }
        )
    )
    exchange = Mock(publish=AsyncMock())
    consumer = OfferEventConsumer(Mock(), exchange, continuation)
    message = make_message()

    await consumer.handle_message(message)

    retried = exchange.publish.await_args.args[0]
    assert retried.headers["x-retry-count"] == 1
    exchange.publish.assert_awaited_once_with(
        retried,
        routing_key=APPROVED_ROUTING_KEY,
    )
    message.ack.assert_awaited_once()
    message.nack.assert_not_awaited()


@pytest.mark.asyncio
async def test_retry_limit_acks_failed_event_without_infinite_requeue() -> None:
    continuation = Mock(
        run=AsyncMock(
            return_value={
                "status": "ERROR",
                "error": "TIMEOUT",
                "retryable": True,
            }
        )
    )
    exchange = Mock(publish=AsyncMock())
    consumer = OfferEventConsumer(Mock(), exchange, continuation)
    message = make_message(retry_count=2)

    await consumer.handle_message(message)

    exchange.publish.assert_not_awaited()
    message.ack.assert_awaited_once()


@pytest.mark.asyncio
async def test_event_consumer_declares_separate_durable_queue_and_bindings() -> None:
    queue = Mock(bind=AsyncMock(), consume=AsyncMock(return_value="consumer-tag"))
    channel = Mock(declare_queue=AsyncMock(return_value=queue))
    consumer = OfferEventConsumer(channel, Mock(), Mock())

    await consumer.start()

    channel.declare_queue.assert_awaited_once_with(EVENT_QUEUE_NAME, durable=True)
    assert queue.bind.await_count == 3
    bound_keys = {call.kwargs["routing_key"] for call in queue.bind.await_args_list}
    assert bound_keys == {
        APPROVED_ROUTING_KEY,
        REJECTED_ROUTING_KEY,
        CONSTRUCTION_DELAY_ROUTING_KEY,
    }
