import json
import logging

import aio_pika
from aio_pika import IncomingMessage
from aio_pika.abc import AbstractChannel, AbstractExchange, AbstractQueue
from pydantic import ValidationError

from app.agents.construction_delay import ConstructionDelayWorkflow
from app.graphs.offer_continuation import OfferContinuationGraph
from app.schemas.events import (
    ConstructionDelayDetectedEvent,
    OfferApprovedEvent,
    OfferRejectedEvent,
)

logger = logging.getLogger(__name__)

EVENT_QUEUE_NAME = "agent-service.events"
APPROVED_ROUTING_KEY = "event.offer.approved"
REJECTED_ROUTING_KEY = "event.offer.rejected"
CONSTRUCTION_DELAY_ROUTING_KEY = "event.construction.delay_detected"
EVENT_ROUTING_KEYS = (
    APPROVED_ROUTING_KEY,
    REJECTED_ROUTING_KEY,
    CONSTRUCTION_DELAY_ROUTING_KEY,
)
RETRY_HEADER = "x-retry-count"
MAX_RETRY_ATTEMPTS = 2


class DomainEventConsumer:
    def __init__(
        self,
        channel: AbstractChannel,
        exchange: AbstractExchange,
        continuation: OfferContinuationGraph,
        construction_workflow: ConstructionDelayWorkflow | None = None,
    ) -> None:
        self._channel = channel
        self._exchange = exchange
        self._continuation = continuation
        self._construction_workflow = construction_workflow
        self._queue: AbstractQueue | None = None
        self._consumer_tag: str | None = None

    async def start(self) -> None:
        self._queue = await self._channel.declare_queue(
            EVENT_QUEUE_NAME,
            durable=True,
        )
        for routing_key in EVENT_ROUTING_KEYS:
            await self._queue.bind(self._exchange, routing_key=routing_key)
        self._consumer_tag = await self._queue.consume(
            self.handle_message,
            no_ack=False,
        )
        logger.info(
            "Domain event consumer started: queue=%s routing_keys=%s",
            EVENT_QUEUE_NAME,
            ",".join(EVENT_ROUTING_KEYS),
        )

    async def stop(self) -> None:
        if self._queue is not None and self._consumer_tag is not None:
            await self._queue.cancel(self._consumer_tag)
        self._consumer_tag = None
        self._queue = None
        logger.info("Domain event consumer stopped")

    async def handle_message(self, message: IncomingMessage) -> None:
        try:
            if message.routing_key == APPROVED_ROUTING_KEY:
                event = OfferApprovedEvent.model_validate_json(message.body)
            elif message.routing_key == REJECTED_ROUTING_KEY:
                event = OfferRejectedEvent.model_validate_json(message.body)
            elif message.routing_key == CONSTRUCTION_DELAY_ROUTING_KEY:
                event = ConstructionDelayDetectedEvent.model_validate_json(message.body)
            else:
                raise ValueError("unsupported event routing key")
            if message.routing_key != f"event.{event.event_type}":
                raise ValueError("event_type does not match routing key")
        except (ValidationError, ValueError, json.JSONDecodeError, UnicodeDecodeError) as exc:
            logger.warning(
                "Rejecting invalid domain event: routing_key=%s error_type=%s",
                message.routing_key,
                type(exc).__name__,
            )
            await message.reject(requeue=False)
            return

        try:
            if isinstance(event, ConstructionDelayDetectedEvent):
                if self._construction_workflow is None:
                    raise RuntimeError("Construction delay workflow is not configured")
                result = await self._construction_workflow.run(
                    building_id=event.payload.building_id,
                    delay_days=event.payload.delay_days,
                    risk_level=event.payload.risk_level,
                )
                subject_id = event.payload.building_id
            else:
                reason = (
                    event.payload.reason
                    if isinstance(event, OfferRejectedEvent)
                    else None
                )
                result = await self._continuation.run(
                    event_type=event.event_type,
                    offer_id=event.payload.offer_id,
                    deal_id=event.payload.deal_id,
                    reason=reason,
                )
                subject_id = event.payload.offer_id
        except Exception:
            logger.exception(
                "Unexpected domain event workflow failure: event_id=%s subject_id=%s",
                event.event_id,
                getattr(event.payload, "offer_id", None)
                or getattr(event.payload, "building_id", None),
            )
            await message.nack(requeue=False)
            return

        if result.get("status") == "ERROR":
            if result.get("retryable") and self._retry_count(message) < MAX_RETRY_ATTEMPTS:
                await self._retry(message)
                return
            logger.error(
                "Domain event workflow failed permanently: event_id=%s "
                "subject_id=%s error=%s",
                event.event_id,
                subject_id,
                result.get("error"),
            )

        await message.ack()

    async def _retry(self, message: IncomingMessage) -> None:
        next_retry = self._retry_count(message) + 1
        headers = dict(message.headers or {})
        headers[RETRY_HEADER] = next_retry
        retry_message = aio_pika.Message(
            body=message.body,
            content_type=message.content_type or "application/json",
            headers=headers,
            delivery_mode=aio_pika.DeliveryMode.PERSISTENT,
        )
        try:
            await self._exchange.publish(
                retry_message,
                routing_key=message.routing_key,
            )
        except Exception:
            logger.exception(
                "Failed to republish domain event retry: routing_key=%s",
                message.routing_key,
            )
            await message.nack(requeue=False)
            return

        await message.ack()
        logger.warning(
            "Domain event scheduled for retry: routing_key=%s retry_count=%d/%d",
            message.routing_key,
            next_retry,
            MAX_RETRY_ATTEMPTS,
        )

    @staticmethod
    def _retry_count(message: IncomingMessage) -> int:
        value = (message.headers or {}).get(RETRY_HEADER, 0)
        try:
            return max(int(value), 0)
        except (TypeError, ValueError):
            return MAX_RETRY_ATTEMPTS


OfferEventConsumer = DomainEventConsumer
