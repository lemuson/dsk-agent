from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

from app.schemas.identifiers import BusinessId, TransportId


class ChatPayload(BaseModel):
    model_config = ConfigDict(extra="forbid")

    user_id: BusinessId
    session_id: BusinessId
    deal_id: BusinessId | None = None
    parking_unit_id: BusinessId | None = None
    storage_unit_id: BusinessId | None = None
    message: str = Field(min_length=1)


class AgentRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: TransportId
    action: Literal["chat"]
    payload: ChatPayload
