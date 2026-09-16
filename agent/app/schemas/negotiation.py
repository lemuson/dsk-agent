from typing import Any

from pydantic import BaseModel, ConfigDict, Field

from app.schemas.identifiers import BusinessId


class DealFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    id: BusinessId
    client_id: BusinessId | None = None
    apartment_id: BusinessId
    stage: str | None = None
    next_action: str | None = None


class ApartmentFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    id: BusinessId
    building_id: BusinessId
    number: str | None = None
    floor: int | None = None
    rooms: int | None = None
    area: float | None = None
    price: int | None = None
    status: str | None = None


class BuildingFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    id: BusinessId
    name: str | None = None
    district: str = Field(min_length=1)
    readiness_percent: int | None = Field(default=None, ge=0, le=100)
    planned_delivery: str | None = None
    forecast_delivery: str | None = None


class CompetitorFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    project_name: str
    district: str
    price_per_sqm: int | None = None
    advantages: str | None = None
    disadvantages: str | None = None


class CompetitorListData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    competitors: list[CompetitorFacts]


class NegotiationContext(BaseModel):
    model_config = ConfigDict(extra="forbid")

    deal: DealFacts
    apartment: ApartmentFacts
    building: BuildingFacts
    competitors: list[CompetitorFacts]


def require_backend_data(data: dict[str, Any] | None) -> dict[str, Any]:
    if data is None:
        raise ValueError("backend response does not contain data")
    return data
