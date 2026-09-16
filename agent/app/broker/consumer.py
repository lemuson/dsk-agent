import json
import logging
from typing import Protocol, cast

import httpx
from aio_pika import IncomingMessage
from aio_pika.abc import AbstractQueue
from gigachat.exceptions import RateLimitError, ServerError
from pydantic import ValidationError

from app.agents.analytics import AnalyticsAgent, AnalyticsIntent
from app.agents.analytics import ClientFactsExtractionError, NoClientMessagesError
from app.agents.dialog import DialogCommandError, DialogService
from app.agents.negotiation import NegotiationAgent, NegotiationIntent
from app.agents.offer import OfferAgent, OfferIntent
from app.agents.router import AgentRouter, RouteDecision
from app.broker.publisher import RpcPublisher
from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.schemas.dialog import (
    DialogAnalyzeRequest,
    DialogAnalyzeResponse,
    DialogReplyAssistRequest,
    DialogReplyAssistResponse,
)
from app.schemas.messages import AgentRequest
from app.schemas.responses import AgentResponse

logger = logging.getLogger(__name__)

RETRY_HEADER = "x-retry-count"
MAX_RETRY_ATTEMPTS = 2
TEMPORARY_LLM_ERRORS = (
    httpx.TransportError,
    TimeoutError,
    RateLimitError,
    ServerError,
)
TEMPORARY_BACKEND_CODES = {"TIMEOUT", "PUBLISH_ERROR", "BACKEND_UNAVAILABLE"}
DIALOG_ANALYZE_ROUTING_KEY = "agent.dialog.analyze"
DIALOG_REPLY_ASSIST_ROUTING_KEY = "agent.dialog.reply_assist"


class ChatGenerator(Protocol):
    async def generate(self, message: str) -> str: ...


class AgentConsumer:
    def __init__(
        self,
        queue: AbstractQueue,
        publisher: RpcPublisher,
        llm: ChatGenerator,
        router: AgentRouter,
        negotiation_agent: NegotiationAgent,
        analytics_agent: AnalyticsAgent,
        offer_agent: OfferAgent | None = None,
        dialog_service: DialogService | None = None,
    ) -> None:
        self._queue = queue
        self._publisher = publisher
        self._llm = llm
        self._router = router
        self._negotiation_agent = negotiation_agent
        self._analytics_agent = analytics_agent
        self._offer_agent = offer_agent
        self._dialog_service = dialog_service
        self._consumer_tag: str | None = None

    async def start(self) -> None:
        self._consumer_tag = await self._queue.consume(self.handle_message, no_ack=False)
        logger.info("Agent consumer started")

    async def stop(self) -> None:
        if self._consumer_tag is not None:
            await self._queue.cancel(self._consumer_tag)
            self._consumer_tag = None
            logger.info("Agent consumer stopped")

    async def handle_message(self, message: IncomingMessage) -> None:
        request_id = self._extract_request_id(message.body)

        if not message.reply_to:
            logger.error(
                "Rejecting RPC request without reply_to: correlation_id=%s",
                message.correlation_id,
            )
            await message.reject(requeue=False)
            return

        if not message.correlation_id or not message.correlation_id.strip():
            logger.error(
                "Rejecting RPC request without correlation_id: reply_to=%s",
                message.reply_to,
            )
            await message.reject(requeue=False)
            return

        if request_id is None:
            logger.error(
                "Rejecting message without a valid request_id: correlation_id=%s",
                message.correlation_id,
            )
            await message.reject(requeue=False)
            return

        try:
            if message.routing_key == DIALOG_ANALYZE_ROUTING_KEY:
                if self._dialog_service is None:
                    raise RuntimeError("Dialog service is not configured")
                request = DialogAnalyzeRequest.model_validate_json(message.body)
                logger.info(
                    "Processing dialog analysis: request_id=%s correlation_id=%s",
                    request.request_id,
                    message.correlation_id,
                )
                result = await self._dialog_service.analyze(request.payload.deal_id)
                response = DialogAnalyzeResponse.ok(request.request_id, result)
            elif message.routing_key == DIALOG_REPLY_ASSIST_ROUTING_KEY:
                if self._dialog_service is None:
                    raise RuntimeError("Dialog service is not configured")
                request = DialogReplyAssistRequest.model_validate_json(message.body)
                logger.info(
                    "Processing dialog reply assist: request_id=%s correlation_id=%s",
                    request.request_id,
                    message.correlation_id,
                )
                result = await self._dialog_service.reply_assist(
                    request.payload.deal_id,
                    request.payload.selected_text,
                )
                response = DialogReplyAssistResponse.ok(request.request_id, result)
            else:
                request = AgentRequest.model_validate_json(message.body)
                logger.info(
                    "Processing chat request: request_id=%s correlation_id=%s",
                    request.request_id,
                    message.correlation_id,
                )
                pending_offer = None
                if self._offer_agent is not None:
                    pending_offer = await self._offer_agent.resume_pending(
                        request.payload.message,
                        request.payload.user_id,
                        request.payload.session_id,
                    )
                if pending_offer is not None:
                    answer, pending_intent = pending_offer
                    decision = RouteDecision(
                        agent="offer",
                        intent=pending_intent,
                        confidence=1.0,
                    )
                else:
                    decision = await self._router.route(request.payload.message)
                    if decision.agent == "negotiation":
                        answer = await self._negotiation_agent.generate(
                            request.payload.message,
                            cast(NegotiationIntent, decision.intent),
                            request.payload.deal_id,
                        )
                    elif decision.agent == "analytics":
                        answer = await self._analytics_agent.generate(
                            request.payload.message,
                            cast(AnalyticsIntent, decision.intent),
                            request.payload.deal_id,
                        )
                    elif decision.agent == "offer" and decision.intent in (
                        "create_offer",
                        "calculate_offer",
                    ):
                        if self._offer_agent is None:
                            raise RuntimeError("Offer agent is not configured")
                        selection = {
                            key: value
                            for key, value in {
                                "parking_unit_id": request.payload.parking_unit_id,
                                "storage_unit_id": request.payload.storage_unit_id,
                            }.items()
                            if value is not None
                        }
                        answer = await self._offer_agent.generate(
                            request.payload.message,
                            cast(OfferIntent, decision.intent),
                            request.payload.deal_id,
                            request.payload.user_id,
                            request.payload.session_id,
                            **selection,
                        )
                    else:
                        answer = await self._llm.generate(request.payload.message)
                response = AgentResponse.ok(
                    request.request_id,
                    answer,
                    decision.agent,
                    decision.intent,
                )
        except (ValidationError, json.JSONDecodeError, UnicodeDecodeError) as exc:
            logger.warning("Invalid request: request_id=%s error=%s", request_id, exc)
            response = self._failure_response(
                message.routing_key,
                request_id,
                "VALIDATION_ERROR",
                str(exc),
            )
        except NoClientMessagesError:
            response = self._failure_response(
                message.routing_key,
                request_id,
                "NO_CLIENT_MESSAGES",
                "В истории сделки нет сообщений клиента для анализа.",
            )
        except ClientFactsExtractionError:
            response = self._failure_response(
                message.routing_key,
                request_id,
                "ANALYSIS_ERROR",
                "Не удалось получить структурированный анализ диалога.",
            )
        except DialogCommandError as exc:
            response = self._failure_response(
                message.routing_key,
                request_id,
                exc.code,
                str(exc),
            )
        except BackendRpcTimeoutError as exc:
            if await self._retry_temporary_error(message, request_id, exc):
                return
            response = self._failure_response(
                message.routing_key,
                request_id,
                "TIMEOUT",
                "Backend request timed out after retry attempts",
            )
        except BackendRpcError as exc:
            if (
                exc.code in TEMPORARY_BACKEND_CODES
                and await self._retry_temporary_error(message, request_id, exc)
            ):
                return
            response = self._failure_response(
                message.routing_key,
                request_id,
                exc.code or "BACKEND_ERROR",
                "Backend request failed",
            )
        except TEMPORARY_LLM_ERRORS as exc:
            if await self._retry_temporary_error(message, request_id, exc):
                return
            response = self._failure_response(
                message.routing_key,
                request_id,
                "INTERNAL_ERROR",
                "Failed to generate a response after temporary errors",
            )
        except Exception:
            logger.exception("Request processing failed: request_id=%s", request_id)
            response = self._failure_response(
                message.routing_key,
                request_id,
                "INTERNAL_ERROR",
                "Failed to generate a response",
            )

        try:
            await self._publisher.publish(
                response=response,
                reply_to=message.reply_to,
                correlation_id=message.correlation_id,
            )
        except Exception:
            logger.exception("RPC response publishing failed: request_id=%s", request_id)
            await message.nack(requeue=True)
            return

        await message.ack()
        logger.info("Request acknowledged: request_id=%s", request_id)

    async def _retry_temporary_error(
        self,
        message: IncomingMessage,
        request_id: str,
        error: Exception,
    ) -> bool:
        retry_count = self._get_retry_count(message)
        if retry_count >= MAX_RETRY_ATTEMPTS:
            logger.error(
                "Temporary processing error retry limit reached: request_id=%s "
                "retry_count=%d error_type=%s",
                request_id,
                retry_count,
                type(error).__name__,
            )
            return False

        next_retry_count = retry_count + 1
        logger.warning(
            "Temporary processing error, scheduling retry: request_id=%s "
            "retry_count=%d/%d error_type=%s",
            request_id,
            next_retry_count,
            MAX_RETRY_ATTEMPTS,
            type(error).__name__,
        )
        try:
            await self._publisher.retry(message, next_retry_count)
        except Exception:
            logger.exception(
                "Failed to republish request for retry: request_id=%s", request_id
            )
            await message.nack(requeue=True)
            return True

        await message.ack()
        return True

    @staticmethod
    def _get_retry_count(message: IncomingMessage) -> int:
        value = (message.headers or {}).get(RETRY_HEADER, 0)
        try:
            retry_count = int(value)
        except (TypeError, ValueError):
            logger.warning("Invalid %s header: %r", RETRY_HEADER, value)
            return MAX_RETRY_ATTEMPTS
        return max(retry_count, 0)

    @staticmethod
    def _failure_response(
        routing_key: str,
        request_id: str,
        code: str,
        message: str,
    ) -> AgentResponse | DialogAnalyzeResponse | DialogReplyAssistResponse:
        if routing_key == DIALOG_ANALYZE_ROUTING_KEY:
            return DialogAnalyzeResponse.fail(request_id, code, message)
        if routing_key == DIALOG_REPLY_ASSIST_ROUTING_KEY:
            return DialogReplyAssistResponse.fail(request_id, code, message)
        return AgentResponse.fail(request_id, code, message)

    @staticmethod
    def _extract_request_id(body: bytes) -> str | None:
        try:
            raw = json.loads(body)
            if not isinstance(raw, dict):
                return None
            request_id = raw.get("request_id")
            if not isinstance(request_id, str):
                return None
            request_id = request_id.strip()
            return request_id or None
        except (json.JSONDecodeError, UnicodeDecodeError, TypeError):
            return None
