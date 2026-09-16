import logging

from pydantic import ValidationError

from app.agents.analytics import AnalyticsAgent
from app.agents.negotiation import NegotiationAgent
from app.schemas.client_facts import DealMessagesData
from app.schemas.dialog import (
    DialogAnalyzeResult,
    DialogReplyAssistResult,
)
from app.schemas.negotiation import DealFacts, require_backend_data
from app.tools.deal import DealTools
from app.tools.messages import MessagesTools

logger = logging.getLogger(__name__)


class DialogCommandError(Exception):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


class DialogService:
    def __init__(
        self,
        analytics_agent: AnalyticsAgent,
        negotiation_agent: NegotiationAgent,
        deal_tools: DealTools,
        messages_tools: MessagesTools,
    ) -> None:
        self._analytics_agent = analytics_agent
        self._negotiation_agent = negotiation_agent
        self._deal_tools = deal_tools
        self._messages_tools = messages_tools

    async def analyze(self, deal_id: int) -> DialogAnalyzeResult:
        try:
            return await self._analytics_agent.analyze_dialog(deal_id)
        except ValidationError as exc:
            raise DialogCommandError(
                "BACKEND_RESPONSE_ERROR",
                "Backend returned invalid dialog data",
            ) from exc

    async def reply_assist(
        self,
        deal_id: int,
        selected_text: str | None,
    ) -> DialogReplyAssistResult:
        try:
            deal_response = await self._deal_tools.get_deal(deal_id)
            deal = DealFacts.model_validate(require_backend_data(deal_response.data))
        except ValidationError as exc:
            raise DialogCommandError(
                "BACKEND_RESPONSE_ERROR",
                "Backend returned invalid deal data",
            ) from exc

        source = "selected_text"
        target_message = selected_text
        if target_message is None:
            try:
                messages_response = await self._messages_tools.get_deal_messages(
                    deal_id,
                    limit=30,
                )
                messages = DealMessagesData.model_validate(
                    require_backend_data(messages_response.data)
                )
            except ValidationError as exc:
                raise DialogCommandError(
                    "BACKEND_RESPONSE_ERROR",
                    "Backend returned invalid dialog history",
                ) from exc

            target_message = next(
                (
                    item.body
                    for item in reversed(messages.messages)
                    if item.direction == "client_to_manager"
                ),
                None,
            )
            if target_message is None:
                raise DialogCommandError(
                    "NO_CLIENT_MESSAGES",
                    "В истории сделки нет сообщений клиента для подготовки ответа.",
                )
            source = "last_client_message"

        try:
            analysis, suggested_reply = await self._negotiation_agent.reply_assist(
                target_message,
                deal,
            )
        except ValidationError as exc:
            raise DialogCommandError(
                "BACKEND_RESPONSE_ERROR",
                "Backend returned invalid factual context",
            ) from exc
        return DialogReplyAssistResult(
            deal_id=deal_id,
            source=source,
            analysis=analysis,
            suggested_reply=suggested_reply,
        )
