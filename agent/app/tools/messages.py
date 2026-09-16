from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class MessagesTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def get_deal_messages(
        self,
        deal_id: int,
        limit: int = 30,
    ) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.deal.messages.get",
            action="deal.messages.get",
            payload={"deal_id": deal_id, "limit": limit},
        )
