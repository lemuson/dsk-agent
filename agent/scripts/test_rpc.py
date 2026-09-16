#!/usr/bin/env python3
"""End-to-end diagnostic client for the Agent Service RabbitMQ RPC flow."""

from __future__ import annotations

import asyncio
import json
import sys
import time
from pathlib import Path
from typing import Any
from uuid import uuid4

import aio_pika
from aio_pika import ExchangeType, IncomingMessage
from aio_pika.abc import AbstractRobustConnection, AbstractRobustQueue

# `python scripts/test_rpc.py` puts scripts/ rather than the project root on sys.path.
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.config import get_settings  # noqa: E402


RESPONSE_TIMEOUT_SECONDS = 70.0
TEST_MESSAGE = "Привет. Чем ты можешь помочь менеджеру по продажам недвижимости?"


async def wait_for_response(
    reply_queue: AbstractRobustQueue,
    correlation_id: str,
) -> bytes:
    """Wait until the private reply queue receives the matching RPC response."""

    loop = asyncio.get_running_loop()
    deadline = loop.time() + RESPONSE_TIMEOUT_SECONDS

    async with reply_queue.iterator() as queue_iterator:
        while True:
            remaining = deadline - loop.time()
            if remaining <= 0:
                raise TimeoutError

            incoming: IncomingMessage = await asyncio.wait_for(
                anext(queue_iterator),
                timeout=remaining,
            )
            async with incoming.process(requeue=False):
                if incoming.correlation_id == correlation_id:
                    return incoming.body


async def run_rpc_test() -> int:
    settings = get_settings()
    request_id = f"req_diagnostic_{uuid4().hex}"
    correlation_id = f"corr_diagnostic_{uuid4().hex}"
    started_at = time.perf_counter()
    connection: AbstractRobustConnection | None = None

    print(f"request_id: {request_id}")
    print(f"correlation_id: {correlation_id}")

    request_payload: dict[str, Any] = {
        "request_id": request_id,
        "action": "chat",
        "payload": {
            "user_id": 15,
            "session_id": 42,
            "deal_id": None,
            "message": TEST_MESSAGE,
        },
    }

    try:
        connection = await aio_pika.connect_robust(
            settings.rabbitmq_url.get_secret_value()
        )
        channel = await connection.channel()

        # Passive declaration verifies that Agent Service already created the exchange.
        exchange = await channel.declare_exchange(
            settings.rabbitmq_exchange,
            ExchangeType.TOPIC,
            durable=True,
            passive=True,
        )
        reply_queue = await channel.declare_queue(
            name="",
            exclusive=True,
            auto_delete=True,
        )

        message = aio_pika.Message(
            body=json.dumps(request_payload, ensure_ascii=False).encode("utf-8"),
            content_type="application/json",
            reply_to=reply_queue.name,
            correlation_id=correlation_id,
            delivery_mode=aio_pika.DeliveryMode.NOT_PERSISTENT,
        )
        await exchange.publish(message, routing_key=settings.rabbitmq_routing_key)

        raw_response = await wait_for_response(reply_queue, correlation_id)
        try:
            response = json.loads(raw_response.decode("utf-8"))
        except (UnicodeDecodeError, json.JSONDecodeError) as exc:
            print(
                f"Ошибка: Agent Service вернул некорректный JSON ({type(exc).__name__}).",
                file=sys.stderr,
            )
            return 1

        print("response:")
        print(json.dumps(response, ensure_ascii=False, indent=2))
        return 0
    except TimeoutError:
        print(
            f"Ошибка: ответ с correlation_id={correlation_id} "
            f"не получен за {RESPONSE_TIMEOUT_SECONDS:.0f} секунд.",
            file=sys.stderr,
        )
        return 1
    except Exception as exc:
        # Do not print connection settings: the RabbitMQ URL may contain credentials.
        print(
            f"Ошибка RPC-теста: {type(exc).__name__}. "
            "Проверьте доступность RabbitMQ и запущенного Agent Service.",
            file=sys.stderr,
        )
        return 1
    finally:
        if connection is not None and not connection.is_closed:
            await connection.close()
        elapsed = time.perf_counter() - started_at
        print(f"elapsed: {elapsed:.2f} s")


if __name__ == "__main__":
    try:
        raise SystemExit(asyncio.run(run_rpc_test()))
    except KeyboardInterrupt:
        print("RPC-тест прерван пользователем.", file=sys.stderr)
        raise SystemExit(130) from None
