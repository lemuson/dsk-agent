import logging
from contextlib import asynccontextmanager
from typing import AsyncIterator

from fastapi import FastAPI, Request, status
from fastapi.responses import JSONResponse

from app.agents.analytics import AnalyticsAgent
from app.agents.construction_delay import ConstructionDelayWorkflow
from app.agents.dialog import DialogService
from app.agents.negotiation import NegotiationAgent
from app.agents.offer import OfferAgent
from app.agents.router import AgentRouter
from app.broker.backend_rpc import BackendRpcClient
from app.broker.connection import RabbitMQConnection
from app.broker.consumer import AgentConsumer
from app.broker.event_consumer import DomainEventConsumer
from app.broker.publisher import RpcPublisher
from app.config import get_settings
from app.llm.gigachat import GigaChatClient
from app.graphs.offer_continuation import OfferContinuationGraph
from app.tools.apartment import ApartmentTools
from app.tools.building import BuildingTools
from app.tools.client import ClientTools
from app.tools.competitor import CompetitorTools
from app.tools.construction import ConstructionTools
from app.tools.deal import DealTools
from app.tools.messages import MessagesTools
from app.tools.offer import OfferTools
from app.tools.recommendation import RecommendationTools


def configure_logging(level: str) -> None:
    logging.basicConfig(
        level=level.upper(),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    settings = get_settings()
    configure_logging(settings.log_level)

    broker = RabbitMQConnection(settings)
    llm = GigaChatClient(settings)
    backend_rpc: BackendRpcClient | None = None
    consumer: AgentConsumer | None = None
    event_consumer: DomainEventConsumer | None = None

    try:
        await broker.connect()
        if broker.channel is None or broker.exchange is None or broker.queue is None:
            raise RuntimeError("RabbitMQ topology was not initialized")

        backend_rpc = BackendRpcClient(broker.channel, broker.exchange)
        await backend_rpc.start()
        await llm.start()

        publisher = RpcPublisher(broker.channel)
        router = AgentRouter(llm)
        negotiation_agent = NegotiationAgent(
            llm,
            DealTools(backend_rpc),
            ApartmentTools(backend_rpc),
            BuildingTools(backend_rpc),
            CompetitorTools(backend_rpc),
        )
        analytics_agent = AnalyticsAgent(
            llm,
            DealTools(backend_rpc),
            ApartmentTools(backend_rpc),
            BuildingTools(backend_rpc),
            ConstructionTools(backend_rpc),
            MessagesTools(backend_rpc),
            ClientTools(backend_rpc),
        )
        offer_agent = OfferAgent(
            llm,
            DealTools(backend_rpc),
            ClientTools(backend_rpc),
            ApartmentTools(backend_rpc),
            OfferTools(backend_rpc),
        )
        dialog_service = DialogService(
            analytics_agent,
            negotiation_agent,
            DealTools(backend_rpc),
            MessagesTools(backend_rpc),
        )
        event_consumer = DomainEventConsumer(
            broker.channel,
            broker.exchange,
            OfferContinuationGraph(OfferTools(backend_rpc)),
            ConstructionDelayWorkflow(
                DealTools(backend_rpc),
                analytics_agent,
                RecommendationTools(backend_rpc),
            ),
        )
        consumer = AgentConsumer(
            broker.queue,
            publisher,
            llm,
            router,
            negotiation_agent,
            analytics_agent,
            offer_agent,
            dialog_service,
        )
        await consumer.start()
        await event_consumer.start()

        app.state.broker = broker
        app.state.backend_rpc = backend_rpc
        yield
    finally:
        if event_consumer is not None:
            await event_consumer.stop()
        if consumer is not None:
            await consumer.stop()
        if backend_rpc is not None:
            await backend_rpc.close()
        await llm.close()
        await broker.close()


app = FastAPI(title="Agent Service", lifespan=lifespan)


@app.get("/health")
async def health(request: Request) -> JSONResponse:
    broker: RabbitMQConnection | None = getattr(request.app.state, "broker", None)
    connected = broker is not None and broker.is_connected
    http_status = status.HTTP_200_OK if connected else status.HTTP_503_SERVICE_UNAVAILABLE
    return JSONResponse(
        status_code=http_status,
        content={
            "status": "ok" if connected else "unavailable",
            "rabbitmq": "connected" if connected else "disconnected",
        },
    )
