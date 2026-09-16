from typing import Any

from pydantic import BaseModel, ConfigDict, Field, model_validator

from app.schemas.identifiers import TransportId


class BackendRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: TransportId
    action: str = Field(min_length=1)
    payload: dict[str, Any]


class BackendError(BaseModel):
    model_config = ConfigDict(extra="forbid")

    code: str = Field(min_length=1)
    message: str = Field(min_length=1)


class BackendResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: TransportId
    success: bool
    data: dict[str, Any] | None
    error: BackendError | None

    @model_validator(mode="after")
    def validate_result(self) -> "BackendResponse":
        if self.success:
            if self.data is None or self.error is not None:
                raise ValueError("successful backend response must contain data and no error")
        elif self.data is not None or self.error is None:
            raise ValueError("failed backend response must contain error and no data")
        return self
