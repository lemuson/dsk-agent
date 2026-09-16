from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class BuildingTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def get_building(self, building_id: int) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.building.get",
            action="building.get",
            payload={"building_id": building_id},
        )
