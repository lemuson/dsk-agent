"""Shared RabbitMQ RPC helper for local diagnostic scripts only."""

from __future__ import annotations

import asyncio
import json
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any
from uuid import uuid4

import aio_pika
from aio_pika import ExchangeType, IncomingMessage
from aio_pika.abc import (
    AbstractExchange,
    AbstractRobustConnection,
    AbstractRobustQueue,
)

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.config import get_settings  # noqa: E402


class DiagnosticRpcError(Exception):
    """Base error raised by the diagnostic RabbitMQ RPC client."""


class DiagnosticRpcTimeoutError(DiagnosticRpcError):
    """Raised when no matching RPC response arrives before the deadline."""


class DiagnosticRpcResponseError(DiagnosticRpcError):
    """Raised when an RPC response cannot be safely interpreted."""


@dataclass(frozen=True)
class DiagnosticRpcResult:
    request_id: str
    correlation_id: str
    response: dict[str, Any]
    elapsed_seconds: float


class DiagnosticRpcClient:
    """One-connection RPC client used by command-line E2E diagnostics."""

    def __init__(self) -> None:
        self._connection: AbstractRobustConnection | None = None
        self._exchange: AbstractExchange | None = None
        self._reply_queue: AbstractRobustQueue | None = None
        self._consumer_tag: str | None = None
        self._pending: dict[str, asyncio.Future[dict[str, Any]]] = {}

    async def __aenter__(self) -> "DiagnosticRpcClient":
        await self.start()
        return self

    async def __aexit__(self, *_: object) -> None:
        await self.close()

    async def start(self) -> None:
        if self._connection is not None:
            return

        settings = get_settings()
        try:
            self._connection = await aio_pika.connect_robust(
                settings.rabbitmq_url.get_secret_value()
            )
            channel = await self._connection.channel()
            self._exchange = await channel.declare_exchange(
                settings.rabbitmq_exchange,
                ExchangeType.TOPIC,
                durable=True,
                passive=True,
            )
            self._reply_queue = await channel.declare_queue(
                name="",
                exclusive=True,
                auto_delete=True,
            )
            self._consumer_tag = await self._reply_queue.consume(
                self._handle_response,
                no_ack=False,
            )
        except Exception:
            await self.close()
            raise

    async def close(self) -> None:
        try:
            if self._reply_queue is not None and self._consumer_tag is not None:
                await self._reply_queue.cancel(self._consumer_tag)
        finally:
            for future in self._pending.values():
                if not future.done():
                    future.cancel()
            self._pending.clear()
            self._consumer_tag = None
            self._reply_queue = None
            self._exchange = None
            if self._connection is not None and not self._connection.is_closed:
                await self._connection.close()
            self._connection = None

    async def call(
        self,
        *,
        routing_key: str,
        action: str,
        payload: dict[str, Any],
        timeout: float = 30.0,
    ) -> DiagnosticRpcResult:
        if self._exchange is None or self._reply_queue is None:
            raise DiagnosticRpcError("Diagnostic RPC client is not started")
        if timeout <= 0:
            raise ValueError("timeout must be greater than zero")

        request_id = f"req_diagnostic_{uuid4().hex}"
        correlation_id = f"corr_diagnostic_{uuid4().hex}"
        request = {
            "request_id": request_id,
            "action": action,
            "payload": payload,
        }
        future = asyncio.get_running_loop().create_future()
        self._pending[correlation_id] = future
        started_at = time.perf_counter()

        message = aio_pika.Message(
            body=json.dumps(request, ensure_ascii=False).encode("utf-8"),
            content_type="application/json",
            reply_to=self._reply_queue.name,
            correlation_id=correlation_id,
            delivery_mode=aio_pika.DeliveryMode.NOT_PERSISTENT,
        )

        try:
            await self._exchange.publish(message, routing_key=routing_key)
            try:
                response = await asyncio.wait_for(
                    asyncio.shield(future),
                    timeout=timeout,
                )
            except TimeoutError as exc:
                raise DiagnosticRpcTimeoutError(
                    f"No response for {routing_key} within {timeout:g} seconds"
                ) from exc

            if response.get("request_id") != request_id:
                raise DiagnosticRpcResponseError(
                    "RPC response request_id does not match the request"
                )
            if not isinstance(response.get("success"), bool):
                raise DiagnosticRpcResponseError(
                    "RPC response does not contain a boolean success field"
                )
            return DiagnosticRpcResult(
                request_id=request_id,
                correlation_id=correlation_id,
                response=response,
                elapsed_seconds=time.perf_counter() - started_at,
            )
        finally:
            pending = self._pending.pop(correlation_id, None)
            if pending is not None and not pending.done():
                pending.cancel()

    async def _handle_response(self, message: IncomingMessage) -> None:
        async with message.process(requeue=False):
            correlation_id = message.correlation_id
            future = self._pending.get(correlation_id) if correlation_id else None
            if future is None or future.done():
                return

            try:
                response = json.loads(message.body.decode("utf-8"))
                if not isinstance(response, dict):
                    raise ValueError("response root must be a JSON object")
            except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
                future.set_exception(
                    DiagnosticRpcResponseError(
                        f"RPC response is not valid JSON: {type(exc).__name__}"
                    )
                )
            else:
                future.set_result(response)


def print_result(result: DiagnosticRpcResult) -> None:
    print(f"request_id: {result.request_id}")
    print(f"correlation_id: {result.correlation_id}")
    print("response:")
    print(json.dumps(result.response, ensure_ascii=False, indent=2))
    print(f"elapsed: {result.elapsed_seconds:.2f} s")


async def run_single_call(
    *,
    routing_key: str,
    action: str,
    payload: dict[str, Any],
    timeout: float,
) -> int:
    try:
        async with DiagnosticRpcClient() as client:
            result = await client.call(
                routing_key=routing_key,
                action=action,
                payload=payload,
                timeout=timeout,
            )
        print_result(result)
        if result.response["success"]:
            return 0
        print("Agent Service returned success=false.", file=sys.stderr)
        return 1
    except DiagnosticRpcTimeoutError as exc:
        print(f"RPC timeout: {exc}", file=sys.stderr)
        return 1
    except DiagnosticRpcResponseError as exc:
        print(f"Invalid RPC response: {exc}", file=sys.stderr)
        return 1
    except Exception as exc:
        # Never print connection settings because the URL may contain credentials.
        print(
            f"RPC diagnostic failed: {type(exc).__name__}. "
            "Check RabbitMQ, Agent Service and environment settings.",
            file=sys.stderr,
        )
        return 1


def positive_timeout(value: str) -> float:
    try:
        timeout = float(value)
    except ValueError as exc:
        raise ValueError("timeout must be a number") from exc
    if timeout <= 0:
        raise ValueError("timeout must be greater than zero")
    return timeout
