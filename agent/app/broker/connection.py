import logging

import aio_pika
from aio_pika import ExchangeType
from aio_pika.abc import (
    AbstractRobustChannel,
    AbstractRobustConnection,
    AbstractRobustExchange,
    AbstractRobustQueue,
)

from app.config import Settings

logger = logging.getLogger(__name__)

DIALOG_ROUTING_KEYS = (
    "agent.dialog.analyze",
    "agent.dialog.reply_assist",
)


class RabbitMQConnection:
    def __init__(self, settings: Settings) -> None:
        self._settings = settings
        self.connection: AbstractRobustConnection | None = None
        self.channel: AbstractRobustChannel | None = None
        self.exchange: AbstractRobustExchange | None = None
        self.queue: AbstractRobustQueue | None = None

    @property
    def is_connected(self) -> bool:
        return self.connection is not None and not self.connection.is_closed

    async def connect(self) -> None:
        logger.info("Connecting to RabbitMQ")
        self.connection = await aio_pika.connect_robust(
            self._settings.rabbitmq_url.get_secret_value()
        )
        self.channel = await self.connection.channel()
        await self.channel.set_qos(prefetch_count=self._settings.rabbitmq_prefetch_count)

        self.exchange = await self.channel.declare_exchange(
            self._settings.rabbitmq_exchange,
            ExchangeType.TOPIC,
            durable=True,
        )
        self.queue = await self.channel.declare_queue(
            self._settings.rabbitmq_queue,
            durable=True,
        )
        await self.queue.bind(
            self.exchange,
            routing_key=self._settings.rabbitmq_routing_key,
        )
        for routing_key in DIALOG_ROUTING_KEYS:
            await self.queue.bind(self.exchange, routing_key=routing_key)
        logger.info(
            "RabbitMQ ready: exchange=%s queue=%s routing_keys=%s",
            self._settings.rabbitmq_exchange,
            self._settings.rabbitmq_queue,
            ",".join((self._settings.rabbitmq_routing_key, *DIALOG_ROUTING_KEYS)),
        )

    async def close(self) -> None:
        if self.connection is not None and not self.connection.is_closed:
            await self.connection.close()
            logger.info("RabbitMQ connection closed")
