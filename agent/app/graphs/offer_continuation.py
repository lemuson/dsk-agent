import logging
from typing import Literal, TypedDict

from langgraph.graph import END, START, StateGraph

from app.broker.backend_rpc import BackendRpcError, BackendRpcTimeoutError
from app.schemas.offer import OfferLookupData, OfferPdfData
from app.schemas.negotiation import require_backend_data
from app.tools.offer import OfferTools

logger = logging.getLogger(__name__)

TEMPORARY_BACKEND_CODES = {"TIMEOUT", "PUBLISH_ERROR", "BACKEND_UNAVAILABLE"}


class OfferContinuationState(TypedDict, total=False):
    event_type: Literal["offer.approved", "offer.rejected"]
    offer_id: int
    deal_id: int
    reason: str
    offer: OfferLookupData
    document_url: str
    status: Literal["DONE", "REJECTED", "SKIPPED", "ERROR"]
    error: str
    retryable: bool


class OfferContinuationGraph:
    def __init__(self, offer_tools: OfferTools) -> None:
        self._offer_tools = offer_tools
        self.compiled = self._build_graph()

    def _build_graph(self):
        graph = StateGraph(OfferContinuationState)
        graph.add_node("mark_rejected", self._mark_rejected)
        graph.add_node("load_offer", self._load_offer)
        graph.add_node("generate_pdf", self._generate_pdf)
        graph.add_conditional_edges(
            START,
            self._route_event,
            {"approved": "load_offer", "rejected": "mark_rejected"},
        )
        graph.add_edge("mark_rejected", END)
        graph.add_conditional_edges(
            "load_offer",
            self._route_after_load,
            {"generate": "generate_pdf", "end": END},
        )
        graph.add_edge("generate_pdf", END)
        return graph.compile()

    async def run(
        self,
        *,
        event_type: Literal["offer.approved", "offer.rejected"],
        offer_id: int,
        deal_id: int,
        reason: str | None = None,
    ) -> OfferContinuationState:
        initial: OfferContinuationState = {
            "event_type": event_type,
            "offer_id": offer_id,
            "deal_id": deal_id,
        }
        if reason is not None:
            initial["reason"] = reason
        return await self.compiled.ainvoke(initial)

    async def _mark_rejected(self, state: OfferContinuationState) -> dict:
        logger.info(
            "Offer continuation rejected: offer_id=%s deal_id=%s reason=%s",
            state["offer_id"],
            state["deal_id"],
            state.get("reason"),
        )
        return {"status": "REJECTED"}

    async def _load_offer(self, state: OfferContinuationState) -> dict:
        try:
            response = await self._offer_tools.get_offer(state["offer_id"])
            offer = OfferLookupData.model_validate(require_backend_data(response.data))
            if offer.id != state["offer_id"] or offer.deal_id != state["deal_id"]:
                raise ValueError("offer response identifiers do not match event")
        except Exception as exc:
            return self._error_state("offer.get", exc)

        if offer.status != "approved":
            logger.warning(
                "Skipping offer PDF because backend status is not approved: "
                "offer_id=%s status=%s",
                offer.id,
                offer.status,
            )
            return {"offer": offer, "status": "SKIPPED"}
        return {"offer": offer}

    async def _generate_pdf(self, state: OfferContinuationState) -> dict:
        try:
            response = await self._offer_tools.generate_offer_pdf(state["offer_id"])
            pdf = OfferPdfData.model_validate(require_backend_data(response.data))
            if pdf.offer_id != state["offer_id"]:
                raise ValueError("PDF response offer_id does not match event")
        except Exception as exc:
            return self._error_state("offer.generate_pdf", exc)

        logger.info(
            "Offer continuation completed: offer_id=%s document_url=%s",
            state["offer_id"],
            pdf.document_url,
        )
        return {"document_url": pdf.document_url, "status": "DONE"}

    @staticmethod
    def _route_event(
        state: OfferContinuationState,
    ) -> Literal["approved", "rejected"]:
        return "approved" if state["event_type"] == "offer.approved" else "rejected"

    @staticmethod
    def _route_after_load(
        state: OfferContinuationState,
    ) -> Literal["generate", "end"]:
        return "end" if state.get("status") in {"SKIPPED", "ERROR"} else "generate"

    @staticmethod
    def _error_state(node: str, exc: Exception) -> dict:
        code = exc.code if isinstance(exc, BackendRpcError) else None
        retryable = isinstance(exc, BackendRpcTimeoutError) or code in TEMPORARY_BACKEND_CODES
        logger.warning(
            "Offer continuation failed: node=%s error_type=%s code=%s retryable=%s",
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
