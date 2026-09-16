from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

from app.schemas.identifiers import BusinessId
from app.schemas.negotiation import ApartmentFacts, BuildingFacts, DealFacts


class ConstructionEventFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    type: str = Field(min_length=1)
    title: str = Field(min_length=1)
    risk_level: str = Field(min_length=1)
    delay_days: int | None = Field(default=None, ge=0)
    completion_percentage: int | None = Field(default=None, ge=0, le=100)


class ConstructionEventsData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    events: list[ConstructionEventFacts]


class AnalyticsContext(BaseModel):
    model_config = ConfigDict(extra="forbid")

    deal: DealFacts
    apartment: ApartmentFacts
    building: BuildingFacts
    construction_events: list[ConstructionEventFacts]
    exact_delivery_date_allowed: bool


class AffectedDeal(BaseModel):
    model_config = ConfigDict(extra="ignore")

    id: BusinessId
    status: Literal["pending", "contract", "completed", "cancelled"]


class AffectedDealsData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    deals: list[AffectedDeal]


class ConstructionDelayContext(BaseModel):
    model_config = ConfigDict(extra="forbid")

    deal: DealFacts
    building_id: BusinessId
    delay_days: int = Field(ge=1)
    risk_level: Literal["low", "medium", "high"]


class RecommendationCreatedData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    recommendation_id: BusinessId
    created: bool
