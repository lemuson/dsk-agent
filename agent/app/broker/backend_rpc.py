import asyncio
import json
import logging
from typing import Any
from uuid import uuid4

import aio_pika
from aio_pika import IncomingMessage
from aio_pika.abc import AbstractChannel, AbstractExchange, AbstractQueue
from pydantic import ValidationError

from app.schemas.backend import BackendResponse, BackendRequest

logger = logging.getLogger(__name__)


class BackendRpcError(Exception):
    def __init__(
        self,
        message: str,
        *,
        code: str | None = None,
        response: BackendResponse | None = None,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.response = response


class BackendRpcTimeoutError(BackendRpcError):
    pass


class BackendResponseValidationError(BackendRpcError):
    pass


class BackendRpcClient:
    def __init__(
        self,
        channel: AbstractChannel,
        exchange: AbstractExchange,
    ) -> None:
        self._channel = channel
        self._exchange = exchange
        self._reply_queue: AbstractQueue | None = None
        self._consumer_tag: str | None = None
        self._pending: dict[str, asyncio.Future[BackendResponse]] = {}

    @property
    def pending_count(self) -> int:
        return len(self._pending)

    async def start(self) -> None:
        if self._reply_queue is not None:
            return

        self._reply_queue = await self._channel.declare_queue(
            name="",
            exclusive=True,
            auto_delete=True,
        )
        self._consumer_tag = await self._reply_queue.consume(
            self._handle_response,
            no_ack=False,
        )
        logger.info("Backend RPC client started")

    async def close(self) -> None:
        try:
            if self._reply_queue is not None and self._consumer_tag is not None:
                await self._reply_queue.cancel(self._consumer_tag)
        finally:
            for future in self._pending.values():
                if not future.done():
                    future.set_exception(
                        BackendRpcError(
                            "Backend RPC client closed before response",
                            code="CLIENT_CLOSED",
                        )
                    )
            self._pending.clear()
            self._consumer_tag = None
            self._reply_queue = None
            logger.info("Backend RPC client closed")

    async def call(
        self,
        routing_key: str,
        action: str,
        payload: dict[str, Any],
        timeout: float = 5.0,
    ) -> BackendResponse:
        if self._reply_queue is None:
            raise BackendRpcError("Backend RPC client is not started", code="NOT_STARTED")
        if timeout <= 0:
            raise ValueError("timeout must be greater than zero")

        request_id = f"req_{uuid4().hex}"
        correlation_id = f"corr_{uuid4().hex}"
        request = BackendRequest(
            request_id=request_id,
            action=action,
            payload=payload,
        )
        future: asyncio.Future[BackendResponse] = (
            asyncio.get_running_loop().create_future()
        )
        self._pending[correlation_id] = future

        message = aio_pika.Message(
            body=request.model_dump_json().encode("utf-8"),
            content_type="application/json",
            reply_to=self._reply_queue.name,
            correlation_id=correlation_id,
            delivery_mode=aio_pika.DeliveryMode.NOT_PERSISTENT,
        )

        try:
            try:
                await self._exchange.publish(message, routing_key=routing_key)
            except Exception as exc:
                raise BackendRpcError(
                    "Failed to publish backend RPC request",
                    code="PUBLISH_ERROR",
                ) from exc

            try:
                response = await asyncio.wait_for(asyncio.shield(future), timeout=timeout)
            except TimeoutError as exc:
                raise BackendRpcTimeoutError(
                    f"Backend RPC request timed out after {timeout:g} seconds",
                    code="TIMEOUT",
                ) from exc

            if response.request_id != request_id:
                raise BackendResponseValidationError(
                    "Backend response request_id does not match the request",
                    code="REQUEST_ID_MISMATCH",
                )
            if not response.success:
                assert response.error is not None
                logger.warning(
                    "Backend RPC failed: routing_key=%s action=%s code=%s "
                    "message=%s request_id=%s",
                    routing_key,
                    action,
                    response.error.code,
                    response.error.message,
                    request_id,
                )
                raise BackendRpcError(
                    response.error.message,
                    code=response.error.code,
                    response=response,
                )
            return response
        finally:
            pending = self._pending.pop(correlation_id, None)
            if pending is not None and not pending.done():
                pending.cancel()

    async def _handle_response(self, message: IncomingMessage) -> None:
        correlation_id = message.correlation_id
        future = self._pending.get(correlation_id) if correlation_id else None

        if future is None:
            logger.warning("Ignoring backend RPC response with unknown correlation_id")
            await message.ack()
            return

        try:
            raw_response = json.loads(message.body)
        except (json.JSONDecodeError, UnicodeDecodeError) as exc:
            if not future.done():
                future.set_exception(
                    BackendResponseValidationError(
                        "Backend response is not valid JSON",
                        code="INVALID_RESPONSE_JSON",
                    )
                )
            logger.warning(
                "Invalid backend RPC response JSON: correlation_id=%s error_type=%s",
                correlation_id,
                type(exc).__name__,
            )
        else:
            try:
                response = BackendResponse.model_validate(raw_response)
            except ValidationError as exc:
                if not future.done():
                    future.set_exception(
                        BackendResponseValidationError(
                            "Backend response does not match the schema",
                            code="INVALID_RESPONSE_SCHEMA",
                        )
                    )
                logger.warning(
                    "Invalid backend RPC response schema: correlation_id=%s error=%s",
                    correlation_id,
                    exc,
                )
            else:
                if not future.done():
                    future.set_result(response)

        await message.ack()
