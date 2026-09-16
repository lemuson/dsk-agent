from typing import Literal, TypedDict

from app.schemas.negotiation import ApartmentFacts, DealFacts
from app.schemas.offer import OfferCalculation, OfferClientFacts


ApprovalStatus = Literal["not_required", "waiting"]


class OfferState(TypedDict, total=False):
    user_id: int
    deal_id: int
    message: str
    intent: Literal["create_offer", "calculate_offer"]
    requested_discount_percent: int | float
    parking_unit_id: int | None
    storage_unit_id: int | None
    deal: DealFacts
    client: OfferClientFacts
    apartment: ApartmentFacts
    calculation: OfferCalculation
    generated_text: str
    offer_id: int
    approval_required: bool
    approval_status: ApprovalStatus
    error: str
    result_message: str
