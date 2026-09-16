from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from app.schemas.client_facts import ClientFacts
from app.schemas.identifiers import BusinessId, TransportId
from app.schemas.responses import ErrorData


class DialogAnalyzePayload(BaseModel):
    model_config = ConfigDict(extra="forbid")

    deal_id: BusinessId
    user_id: BusinessId


class DialogAnalyzeRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: TransportId
    action: Literal["dialog.analyze"]
    payload: DialogAnalyzePayload


class DialogAnalyzeResult(BaseModel):
    model_config = ConfigDict(extra="forbid")

    deal_id: BusinessId
    client_id: BusinessId
    analysis: ClientFacts
    preferences_updated: bool


class DialogAnalyzeResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: TransportId
    success: bool
    data: DialogAnalyzeResult | None
    error: ErrorData | None

    @classmethod
    def ok(cls, request_id: str, data: DialogAnalyzeResult) -> "DialogAnalyzeResponse":
        return cls(request_id=request_id, success=True, data=data, error=None)

    @classmethod
    def fail(cls, request_id: str, code: str, message: str) -> "DialogAnalyzeResponse":
        return cls(
            request_id=request_id,
            success=False,
            data=None,
            error=ErrorData(code=code, message=message),
        )


class DialogReplyAssistPayload(BaseModel):
    model_config = ConfigDict(extra="forbid")

    deal_id: BusinessId
    user_id: BusinessId
    selected_text: str | None = None

    @field_validator("selected_text", mode="before")
    @classmethod
    def normalize_selected_text(cls, value: object) -> object:
        if isinstance(value, str):
            normalized = value.strip()
            return normalized or None
        return value


class DialogReplyAssistRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: TransportId
    action: Literal["dialog.reply_assist"]
    payload: DialogReplyAssistPayload


ReplyAssistIntent = Literal[
    "price_objection",
    "competitor_comparison",
    "timing_objection",
    "property_objection",
    "general_question",
    "other_objection",
    "unknown",
]


class ReplyAssistAnalysis(BaseModel):
    model_config = ConfigDict(extra="forbid", str_strip_whitespace=True)

    intent: ReplyAssistIntent
    summary: str = Field(min_length=1)


class DialogReplyAssistResult(BaseModel):
    model_config = ConfigDict(extra="forbid")

    deal_id: BusinessId
    source: Literal["last_client_message", "selected_text"]
    analysis: ReplyAssistAnalysis
    suggested_reply: str = Field(min_length=1)


class DialogReplyAssistResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: TransportId
    success: bool
    data: DialogReplyAssistResult | None
    error: ErrorData | None

    @classmethod
    def ok(
        cls,
        request_id: str,
        data: DialogReplyAssistResult,
    ) -> "DialogReplyAssistResponse":
        return cls(request_id=request_id, success=True, data=data, error=None)

    @classmethod
    def fail(
        cls,
        request_id: str,
        code: str,
        message: str,
    ) -> "DialogReplyAssistResponse":
        return cls(
            request_id=request_id,
            success=False,
            data=None,
            error=ErrorData(code=code, message=message),
        )
