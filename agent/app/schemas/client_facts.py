from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

from app.schemas.identifiers import BusinessId


class DealMessage(BaseModel):
    model_config = ConfigDict(extra="ignore", str_strip_whitespace=True)

    id: BusinessId
    direction: Literal["client_to_manager", "manager_to_client"]
    body: str = Field(min_length=1)


class DealMessagesData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    messages: list[DealMessage]


class ClientFacts(BaseModel):
    model_config = ConfigDict(extra="forbid", str_strip_whitespace=True)

                                                                               
                                                                             
                                                                             
    budget_min: int | None = Field(ge=0)
    budget_max: int | None = Field(ge=0)
    rooms: int | None = Field(ge=1)
    floor_min: int | None = Field(ge=1)
    floor_max: int | None = Field(ge=1)
    parking_required: bool | None
    renovation_required: bool | None
    preferred_district: str | None
    purchase_timeline: str | None
    important_factors: list[str]
    objections: list[str]
    summary: str = Field(min_length=1)

    @model_validator(mode="after")
    def validate_ranges(self) -> "ClientFacts":
        if (
            self.budget_min is not None
            and self.budget_max is not None
            and self.budget_min > self.budget_max
        ):
            raise ValueError("budget_min must not exceed budget_max")
        if (
            self.floor_min is not None
            and self.floor_max is not None
            and self.floor_min > self.floor_max
        ):
            raise ValueError("floor_min must not exceed floor_max")
        return self
