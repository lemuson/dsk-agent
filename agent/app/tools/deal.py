from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class DealTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def get_deal(self, deal_id: int) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.deal.get",
            action="deal.get",
            payload={"deal_id": deal_id},
        )

    async def list_deals_by_building(self, building_id: int) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.deal.list_by_building",
            action="deal.list_by_building",
            payload={"building_id": building_id},
        )
