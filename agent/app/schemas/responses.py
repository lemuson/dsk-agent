from pydantic import BaseModel, ConfigDict

from app.agents.router import AgentName, IntentName
from app.schemas.identifiers import TransportId


class ChatResponseData(BaseModel):
    model_config = ConfigDict(extra="forbid")

    message: str
    agent: AgentName
    intent: IntentName


class ErrorData(BaseModel):
    model_config = ConfigDict(extra="forbid")

    code: str
    message: str


class AgentResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: TransportId
    success: bool
    data: ChatResponseData | None
    error: ErrorData | None

    @classmethod
    def ok(
        cls,
        request_id: str,
        message: str,
        agent: AgentName,
        intent: IntentName,
    ) -> "AgentResponse":
        return cls(
            request_id=request_id,
            success=True,
            data=ChatResponseData(message=message, agent=agent, intent=intent),
            error=None,
        )

    @classmethod
    def fail(cls, request_id: str, code: str, message: str) -> "AgentResponse":
        return cls(
            request_id=request_id,
            success=False,
            data=None,
            error=ErrorData(code=code, message=message),
        )
