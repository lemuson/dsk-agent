from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class OfferTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def calculate_offer(
        self,
        deal_id: int,
        requested_by: int,
        discount_percent: int | float,
        parking_unit_id: int | None = None,
        storage_unit_id: int | None = None,
    ) -> BackendResponse:
        payload = {
            "deal_id": deal_id,
            "requested_by": requested_by,
            "discount_percent": discount_percent,
        }
        if parking_unit_id is not None:
            payload["parking_unit_id"] = parking_unit_id
        if storage_unit_id is not None:
            payload["storage_unit_id"] = storage_unit_id
        return await self._rpc.call(
            routing_key="backend.offer.calculate",
            action="offer.calculate",
            payload=payload,
        )

    async def get_offer(self, offer_id: int) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.offer.get",
            action="offer.get",
            payload={"offer_id": offer_id},
        )

    async def create_offer(
        self,
        deal_id: int,
        created_by: int,
        discount_percent: int | float,
        generated_text: str,
        parking_unit_id: int | None = None,
        storage_unit_id: int | None = None,
    ) -> BackendResponse:
        payload = {
            "deal_id": deal_id,
            "created_by": created_by,
            "discount_percent": discount_percent,
            "generated_text": generated_text,
        }
        if parking_unit_id is not None:
            payload["parking_unit_id"] = parking_unit_id
        if storage_unit_id is not None:
            payload["storage_unit_id"] = storage_unit_id
        return await self._rpc.call(
            routing_key="backend.offer.create",
            action="offer.create",
            payload=payload,
        )

    async def request_offer_approval(
        self,
        offer_id: int,
        requested_by: int,
    ) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.offer.request_approval",
            action="offer.request_approval",
            payload={
                "offer_id": offer_id,
                "requested_by": requested_by,
            },
        )

    async def generate_offer_pdf(self, offer_id: int) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.offer.generate_pdf",
            action="offer.generate_pdf",
            payload={"offer_id": offer_id},
        )
