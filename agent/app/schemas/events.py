from datetime import datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field

from app.schemas.identifiers import BusinessId, TransportId


class OfferApprovedPayload(BaseModel):
    model_config = ConfigDict(extra="forbid")

    offer_id: BusinessId
    deal_id: BusinessId


class OfferRejectedPayload(OfferApprovedPayload):
    reason: str = Field(min_length=1)


class OfferApprovedEvent(BaseModel):
    model_config = ConfigDict(extra="forbid")

    event_id: TransportId
    event_type: Literal["offer.approved"]
    occurred_at: datetime
    payload: OfferApprovedPayload


class OfferRejectedEvent(BaseModel):
    model_config = ConfigDict(extra="forbid")

    event_id: TransportId
    event_type: Literal["offer.rejected"]
    occurred_at: datetime
    payload: OfferRejectedPayload


class ConstructionDelayPayload(BaseModel):
    model_config = ConfigDict(extra="forbid")

    building_id: BusinessId
    delay_days: int = Field(ge=1)
    risk_level: Literal["low", "medium", "high"]


class ConstructionDelayDetectedEvent(BaseModel):
    model_config = ConfigDict(extra="forbid")

    event_id: TransportId
    event_type: Literal["construction.delay_detected"]
    occurred_at: datetime
    payload: ConstructionDelayPayload
