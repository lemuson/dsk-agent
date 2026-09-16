from decimal import Decimal
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, model_validator

from app.schemas.identifiers import BusinessId
from app.schemas.negotiation import ApartmentFacts, DealFacts


class OfferRequestFacts(BaseModel):
    """Only discount facts explicitly stated in the manager's request."""

    model_config = ConfigDict(extra="forbid")

    discount_mentioned: bool
    requested_discount_percent: Decimal | None = Field(
        ge=Decimal("0"),
        le=Decimal("100"),
        decimal_places=2,
    )

    @model_validator(mode="after")
    def validate_discount_presence(self) -> "OfferRequestFacts":
        if not self.discount_mentioned and self.requested_discount_percent is not None:
            raise ValueError(
                "requested_discount_percent requires discount_mentioned=true"
            )
        return self


class OfferClientFacts(BaseModel):
    model_config = ConfigDict(extra="ignore")

    id: BusinessId
    full_name: str | None = None
    budget_max: int | None = Field(default=None, ge=0)
    preferences: dict[str, Any] = Field(default_factory=dict)


class OfferCalculation(BaseModel):
    model_config = ConfigDict(extra="ignore")

    deal_id: BusinessId
    base_price: int = Field(ge=0)
    apartment_price: int | None = Field(default=None, ge=0)
    parking_unit_id: BusinessId | None = None
    parking_number: str | None = None
    parking_price: int = Field(default=0, ge=0)
    storage_unit_id: BusinessId | None = None
    storage_number: str | None = None
    storage_price: int = Field(default=0, ge=0)
    discount_percent: float = Field(ge=0)
    discount_amount: int = Field(ge=0)
    final_price: int = Field(ge=0)
    max_allowed_discount: float = Field(ge=0)
    requires_approval: bool

    @model_validator(mode="after")
    def derive_apartment_price(self) -> "OfferCalculation":
        if self.apartment_price is None:
            self.apartment_price = max(
                0,
                self.base_price - self.parking_price - self.storage_price,
            )
        return self


class OfferContext(BaseModel):
    model_config = ConfigDict(extra="forbid")

    deal: DealFacts
    client: OfferClientFacts
    apartment: ApartmentFacts
    calculation: OfferCalculation


class OfferCreatedData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    offer_id: BusinessId
    status: str = Field(min_length=1)


class OfferApprovalData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    offer_id: BusinessId
    status: str = Field(min_length=1)


class OfferLookupData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    id: BusinessId
    deal_id: BusinessId
    status: Literal["draft", "pending_approval", "approved", "rejected"]


class OfferPdfData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    offer_id: BusinessId
    document_url: str = Field(min_length=1)
