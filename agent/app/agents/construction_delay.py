import logging
from typing import Literal, TypedDict

import httpx
from gigachat.exceptions import RateLimitError, ServerError
from app.agents.analytics import AnalyticsAgent
from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.schemas.analytics import AffectedDealsData, RecommendationCreatedData
from app.schemas.negotiation import DealFacts, require_backend_data
from app.tools.deal import DealTools
from app.tools.recommendation import RecommendationTools

logger = logging.getLogger(__name__)

TEMPORARY_BACKEND_CODES = {"TIMEOUT", "PUBLISH_ERROR", "BACKEND_UNAVAILABLE"}
TEMPORARY_LLM_ERRORS = (
    httpx.TransportError,
    TimeoutError,
    RateLimitError,
    ServerError,
)


class ConstructionDelayResult(TypedDict, total=False):
    status: Literal["DONE", "PARTIAL", "ERROR"]
    recommendations_created: int
    error: str
    retryable: bool


class ConstructionDelayWorkflow:
    def __init__(
        self,
        deal_tools: DealTools,
        analytics_agent: AnalyticsAgent,
        recommendation_tools: RecommendationTools,
    ) -> None:
        self._deal_tools = deal_tools
        self._analytics_agent = analytics_agent
        self._recommendation_tools = recommendation_tools

    async def run(
        self,
        *,
        building_id: int,
        delay_days: int,
        risk_level: Literal["low", "medium", "high"],
    ) -> ConstructionDelayResult:
        try:
            response = await self._deal_tools.list_deals_by_building(building_id)
            affected = AffectedDealsData.model_validate(
                require_backend_data(response.data)
            )
        except Exception as exc:
            return self._error_result("deal.list_by_building", exc)

        created = 0
        had_permanent_error = False
        for affected_deal in affected.deals:
            if affected_deal.status not in {"pending", "contract"}:
                continue
            try:
                response = await self._deal_tools.get_deal(affected_deal.id)
                deal = DealFacts.model_validate(require_backend_data(response.data))
                if deal.id != affected_deal.id:
                    raise ValueError("deal response id does not match affected deal")

                recommendation = (
                    await self._analytics_agent.generate_delay_recommendation(
                        deal,
                        building_id=building_id,
                        delay_days=delay_days,
                        risk_level=risk_level,
                    )
                )
                create_response = (
                    await self._recommendation_tools.create_recommendation(
                        deal.id,
                        "construction_risk",
                        recommendation,
                    )
                )
                created_data = RecommendationCreatedData.model_validate(
                    require_backend_data(create_response.data)
                )
                if not created_data.created:
                    raise ValueError("backend did not confirm recommendation creation")
                created += 1
            except Exception as exc:
                error = self._error_result(
                    f"recommendation for deal {affected_deal.id}", exc
                )
                if error["retryable"]:
                    error["recommendations_created"] = created
                    return error
                had_permanent_error = True

        return {
            "status": "PARTIAL" if had_permanent_error else "DONE",
            "recommendations_created": created,
            "retryable": False,
        }

    @staticmethod
    def _error_result(node: str, exc: Exception) -> ConstructionDelayResult:
        code = exc.code if isinstance(exc, BackendRpcError) else None
        retryable = (
            isinstance(exc, BackendRpcTimeoutError)
            or isinstance(exc, TEMPORARY_LLM_ERRORS)
            or code in TEMPORARY_BACKEND_CODES
        )
        logger.warning(
            "Construction delay workflow failed: node=%s error_type=%s "
            "code=%s retryable=%s",
            node,
            type(exc).__name__,
            code,
            retryable,
        )
        return {
            "status": "ERROR",
            "error": code or type(exc).__name__,
            "retryable": retryable,
        }
