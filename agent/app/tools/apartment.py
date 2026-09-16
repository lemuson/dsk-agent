from app.broker.backend_rpc import BackendRpcClient
from app.schemas.backend import BackendResponse


class ApartmentTools:
    def __init__(self, rpc: BackendRpcClient) -> None:
        self._rpc = rpc

    async def get_apartment(self, apartment_id: int) -> BackendResponse:
        return await self._rpc.call(
            routing_key="backend.apartment.get",
            action="apartment.get",
            payload={"apartment_id": apartment_id},
        )
