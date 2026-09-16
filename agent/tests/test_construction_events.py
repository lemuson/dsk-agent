import json
from unittest.mock import AsyncMock, Mock

import httpx
import pytest
from pydantic import ValidationError

from app.agents.analytics import CONSTRUCTION_DELAY_SYSTEM_PROMPT, AnalyticsAgent
from app.agents.construction_delay import ConstructionDelayWorkflow
from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.broker.event_consumer import (
    CONSTRUCTION_DELAY_ROUTING_KEY,
    DomainEventConsumer,
)
from app.schemas.backend import BackendResponse
from app.schemas.events import ConstructionDelayDetectedEvent
from app.schemas.negotiation import DealFacts


BUILDING_ID = 401
DEAL_ID = 101


def backend_response(data: dict) -> BackendResponse:
    return BackendResponse(
        request_id="99999999-9999-9999-9999-999999999999",
        success=True,
        data=data,
        error=None,
    )


def event_body() -> dict:
    return {
        "event_id": "event_construction_delay_123",
        "event_type": "construction.delay_detected",
        "occurred_at": "2026-09-08T14:30:00Z",
        "payload": {
            "building_id": BUILDING_ID,
            "delay_days": 21,
            "risk_level": "high",
        },
    }


def make_message(body: dict | None = None) -> Mock:
    message = Mock()
    message.body = json.dumps(body or event_body()).encode()
    message.routing_key = CONSTRUCTION_DELAY_ROUTING_KEY
    message.content_type = "application/json"
    message.headers = {}
    message.ack = AsyncMock()
    message.nack = AsyncMock()
    message.reject = AsyncMock()
    return message


def make_workflow() -> tuple[ConstructionDelayWorkflow, Mock, Mock, Mock]:
    deal_tools = Mock(
        list_deals_by_building=AsyncMock(
            return_value=backend_response(
                {
                    "deals": [
                        {"id": DEAL_ID, "status": "pending"},
                        {
                            "id": 102,
                            "status": "completed",
                        },
                    ]
                }
            )
        ),
        get_deal=AsyncMock(
            return_value=backend_response(
                {
                    "id": DEAL_ID,
                    "client_id": 201,
                    "apartment_id": 301,
                    "stage": "negotiation",
                }
            )
        ),
    )
    analytics = Mock(
        generate_delay_recommendation=AsyncMock(
            return_value="Сообщите клиенту о риске задержки и уточните его планы."
        )
    )
    recommendation_tools = Mock(
        create_recommendation=AsyncMock(
            return_value=backend_response(
                {
                    "recommendation_id": 701,
                    "created": True,
                }
            )
        )
    )
    return (
        ConstructionDelayWorkflow(deal_tools, analytics, recommendation_tools),
        deal_tools,
        analytics,
        recommendation_tools,
    )


def test_construction_delay_event_schema_is_strict() -> None:
    event = ConstructionDelayDetectedEvent.model_validate(event_body())
    assert event.payload.building_id == BUILDING_ID
    assert event.payload.delay_days == 21
    assert event.payload.risk_level == "high"

    invalid = event_body()
    invalid["payload"]["risk_level"] = "critical"
    with pytest.raises(ValidationError):
        ConstructionDelayDetectedEvent.model_validate(invalid)


@pytest.mark.asyncio
async def test_analytics_recommendation_uses_only_factual_context() -> None:
    llm = Mock(generate=AsyncMock(return_value="Рекомендация"))
    analytics = AnalyticsAgent(llm, Mock(), Mock(), Mock(), Mock(), Mock(), Mock())
    deal = DealFacts.model_validate(
        {
            "id": DEAL_ID,
            "client_id": 201,
            "apartment_id": 301,
            "stage": "negotiation",
        }
    )

    result = await analytics.generate_delay_recommendation(
        deal,
        building_id=BUILDING_ID,
        delay_days=21,
        risk_level="high",
    )

    assert result == "Рекомендация"
    prompt = llm.generate.await_args.args[0]
    assert str(DEAL_ID) in prompt
    assert str(BUILDING_ID) in prompt
    assert '"delay_days": 21' in prompt
    assert '"risk_level": "high"' in prompt
    assert '"request_id"' not in prompt
    assert llm.generate.await_args.kwargs["system_prompt"] == (
        CONSTRUCTION_DELAY_SYSTEM_PROMPT
    )


@pytest.mark.asyncio
async def test_workflow_creates_recommendation_only_for_open_backend_deal() -> None:
    workflow, deal_tools, analytics, recommendation_tools = make_workflow()

    result = await workflow.run(
        building_id=BUILDING_ID,
        delay_days=21,
        risk_level="high",
    )

    deal_tools.list_deals_by_building.assert_awaited_once_with(BUILDING_ID)
    deal_tools.get_deal.assert_awaited_once_with(DEAL_ID)
    analytics.generate_delay_recommendation.assert_awaited_once()
    assert analytics.generate_delay_recommendation.await_args.kwargs == {
        "building_id": BUILDING_ID,
        "delay_days": 21,
        "risk_level": "high",
    }
    recommendation_tools.create_recommendation.assert_awaited_once_with(
        DEAL_ID,
        "construction_risk",
        "Сообщите клиенту о риске задержки и уточните его планы.",
    )
    assert result["status"] == "DONE"
    assert result["recommendations_created"] == 1


@pytest.mark.asyncio
async def test_temporary_backend_error_is_retryable() -> None:
    workflow, deal_tools, analytics, recommendation_tools = make_workflow()
    deal_tools.list_deals_by_building.side_effect = BackendRpcTimeoutError(
        "timeout",
        code="TIMEOUT",
    )

    result = await workflow.run(
        building_id=BUILDING_ID,
        delay_days=21,
        risk_level="high",
    )

    assert result["status"] == "ERROR"
    assert result["retryable"] is True
    analytics.generate_delay_recommendation.assert_not_awaited()
    recommendation_tools.create_recommendation.assert_not_awaited()


@pytest.mark.asyncio
async def test_temporary_gigachat_error_is_retryable() -> None:
    workflow, _, analytics, recommendation_tools = make_workflow()
    analytics.generate_delay_recommendation.side_effect = httpx.ConnectError(
        "network error"
    )

    result = await workflow.run(
        building_id=BUILDING_ID,
        delay_days=21,
        risk_level="high",
    )

    assert result["status"] == "ERROR"
    assert result["retryable"] is True
    recommendation_tools.create_recommendation.assert_not_awaited()


@pytest.mark.asyncio
async def test_permanent_deal_error_does_not_create_fake_recommendation() -> None:
    workflow, deal_tools, analytics, recommendation_tools = make_workflow()
    deal_tools.get_deal.side_effect = BackendRpcError(
        "not found",
        code="DEAL_NOT_FOUND",
    )

    result = await workflow.run(
        building_id=BUILDING_ID,
        delay_days=21,
        risk_level="high",
    )

    assert result["status"] == "PARTIAL"
    assert result["retryable"] is False
    analytics.generate_delay_recommendation.assert_not_awaited()
    recommendation_tools.create_recommendation.assert_not_awaited()


@pytest.mark.asyncio
async def test_event_consumer_routes_construction_delay() -> None:
    construction_workflow = Mock(
        run=AsyncMock(
            return_value={
                "status": "DONE",
                "recommendations_created": 1,
                "retryable": False,
            }
        )
    )
    consumer = DomainEventConsumer(Mock(), Mock(), Mock(), construction_workflow)
    message = make_message()

    await consumer.handle_message(message)

    construction_workflow.run.assert_awaited_once_with(
        building_id=BUILDING_ID,
        delay_days=21,
        risk_level="high",
    )
    message.ack.assert_awaited_once()


@pytest.mark.asyncio
async def test_invalid_construction_event_is_rejected() -> None:
    invalid = event_body()
    invalid["payload"]["delay_days"] = 0
    construction_workflow = Mock(run=AsyncMock())
    consumer = DomainEventConsumer(Mock(), Mock(), Mock(), construction_workflow)
    message = make_message(invalid)

    await consumer.handle_message(message)

    message.reject.assert_awaited_once_with(requeue=False)
    construction_workflow.run.assert_not_awaited()
