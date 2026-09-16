from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class CompetitorTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def list_competitors(self, district: str) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.competitor.list",
            action="competitor.list",
            payload={"district": district},
        )
