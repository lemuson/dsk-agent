from typing import Any
from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class ClientTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def get_client(self, client_id: int) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.client.get",
            action="client.get",
            payload={"client_id": client_id},
        )

    async def update_client_preferences(
        self,
        client_id: int,
        *,
        preferences: dict[str, Any],
        budget_max: int | None = None,
    ) -> BackendResponse:
        payload: dict[str, Any] = {
            "client_id": client_id,
            "preferences": preferences,
        }
        if budget_max is not None:
            payload["budget_max"] = budget_max

        return await self._rpc.call(
            routing_key="backend.client.update_preferences",
            action="client.update_preferences",
            payload=payload,
        )
