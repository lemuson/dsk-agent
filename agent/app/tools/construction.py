from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class ConstructionTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def get_construction_events(self, building_id: int) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.construction.events.get",
            action="construction.events.get",
            payload={"building_id": building_id},
        )
