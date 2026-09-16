import aio_pika
from aio_pika import IncomingMessage
from aio_pika.abc import AbstractChannel
from pydantic import BaseModel


class RpcPublisher:
    def __init__(self, channel: AbstractChannel) -> None:
        self._channel = channel

    async def publish(
        self,
        response: BaseModel,
        reply_to: str,
        correlation_id: str | None,
    ) -> None:
        message = aio_pika.Message(
            body=response.model_dump_json().encode("utf-8"),
            content_type="application/json",
            correlation_id=correlation_id,
            delivery_mode=aio_pika.DeliveryMode.NOT_PERSISTENT,
        )
        await self._channel.default_exchange.publish(message, routing_key=reply_to)

    async def retry(self, message: IncomingMessage, retry_count: int) -> None:
        headers = dict(message.headers or {})
        headers["x-retry-count"] = retry_count

        retry_message = aio_pika.Message(
            body=message.body,
            headers=headers,
            content_type=message.content_type,
            content_encoding=message.content_encoding,
            delivery_mode=message.delivery_mode,
            priority=message.priority,
            correlation_id=message.correlation_id,
            reply_to=message.reply_to,
            message_id=message.message_id,
            type=message.type,
            user_id=message.user_id,
            app_id=message.app_id,
        )
        exchange = await self._channel.get_exchange(message.exchange)
        await exchange.publish(retry_message, routing_key=message.routing_key)
