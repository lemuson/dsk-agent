from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class RecommendationTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def create_recommendation(
        self,
        deal_id: int,
        kind: str,
        recommendation: str,
    ) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.recommendation.create",
            action="recommendation.create",
            payload={
                "deal_id": deal_id,
                "kind": kind,
                "recommendation": recommendation,
            },
        )
