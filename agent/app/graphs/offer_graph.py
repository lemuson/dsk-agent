import logging
from typing import Literal, Protocol, TypeVar

from langgraph.graph import END, START, StateGraph
from pydantic import BaseModel

from app.broker.backend_rpc import BackendRpcError
from app.graphs.offer_state import OfferState
from app.schemas.negotiation import ApartmentFacts, DealFacts, require_backend_data
from app.schemas.offer import (
    OfferApprovalData,
    OfferCalculation,
    OfferClientFacts,
    OfferContext,
    OfferCreatedData,
)
from app.tools.apartment import ApartmentTools
from app.tools.client import ClientTools
from app.tools.deal import DealTools
from app.tools.offer import OfferTools

logger = logging.getLogger(__name__)
OfferModelT = TypeVar("OfferModelT", bound=BaseModel)

BACKEND_DATA_ERROR_RESPONSE = (
    "Не удалось получить фактические данные для формирования коммерческого предложения."
)
OFFER_GENERATION_ERROR_RESPONSE = "Не удалось подготовить текст коммерческого предложения."
OFFER_SAVE_ERROR_RESPONSE = "Сохранить коммерческое предложение не удалось."
APPROVAL_REQUEST_ERROR_RESPONSE = (
    "Предложение подготовлено, но запрос на согласование отправить не удалось."
)
OFFER_SYSTEM_PROMPT = """Ты помощник менеджера по продажам недвижимости.
Сформируй короткий персональный текст коммерческого предложения.

Строгие правила:
- Используй только данные из секции «Фактический контекст backend».
- Все цены, скидки и суммы бери дословно из calculation.
- ЗАПРЕЩЕНО пересчитывать цену или менять discount.
- ЗАПРЕЩЕНО придумывать стоимость и условия сделки.
- ЗАПРЕЩЕНО придумывать характеристики квартиры или данные клиента.
- Не добавляй преимущества, которых нет в factual context.
- Не упоминай внутренние технические поля и backend.

Ответ должен быть кратким, понятным клиенту и готовым к отправке менеджером.
"""


class OfferLlmClient(Protocol):
    async def generate(
        self,
        message: str,
        *,
        system_prompt: str | None = None,
    ) -> str: ...

    async def parse_structured(
        self,
        prompt: str,
        response_model: type[OfferModelT],
    ) -> OfferModelT: ...


class OfferGraph:
    def __init__(
        self,
        llm: OfferLlmClient,
        deal_tools: DealTools,
        client_tools: ClientTools,
        apartment_tools: ApartmentTools,
        offer_tools: OfferTools,
    ) -> None:
        self._llm = llm
        self._deal_tools = deal_tools
        self._client_tools = client_tools
        self._apartment_tools = apartment_tools
        self._offer_tools = offer_tools
        self.compiled = self._build_graph()

    def _build_graph(self):
        graph = StateGraph(OfferState)
        graph.add_node("load_deal", self._load_deal)
        graph.add_node("load_client", self._load_client)
        graph.add_node("load_apartment", self._load_apartment)
        graph.add_node("calculate_offer", self._calculate_offer)
        graph.add_node("generate_text", self._generate_text)
        graph.add_node("create_offer", self._create_offer)
        graph.add_node("request_approval", self._request_approval)

        graph.add_edge(START, "load_deal")
        graph.add_conditional_edges(
            "load_deal",
            self._route_error,
            {"next": "load_client", "end": END},
        )
        graph.add_conditional_edges(
            "load_client",
            self._route_error,
            {"next": "load_apartment", "end": END},
        )
        graph.add_conditional_edges(
            "load_apartment",
            self._route_error,
            {"next": "calculate_offer", "end": END},
        )
        graph.add_conditional_edges(
            "calculate_offer",
            self._route_after_calculation,
            {"generate": "generate_text", "end": END},
        )
        graph.add_conditional_edges(
            "generate_text",
            self._route_error,
            {"next": "create_offer", "end": END},
        )
        graph.add_conditional_edges(
            "create_offer",
            self._route_after_create,
            {"approval": "request_approval", "end": END},
        )
        graph.add_edge("request_approval", END)
        return graph.compile()

    async def run(self, state: OfferState) -> OfferState:
        return await self.compiled.ainvoke(state)

    async def _load_deal(self, state: OfferState) -> dict:
        try:
            response = await self._deal_tools.get_deal(state["deal_id"])
            deal = DealFacts.model_validate(require_backend_data(response.data))
            if deal.client_id is None:
                raise ValueError("deal does not contain client_id")
            return {"deal": deal}
        except Exception as exc:
            return self._backend_error("load_deal", exc)

    async def _load_client(self, state: OfferState) -> dict:
        try:
            client_id = state["deal"].client_id
            if client_id is None:
                raise ValueError("deal does not contain client_id")
            response = await self._client_tools.get_client(client_id)
            client = OfferClientFacts.model_validate(require_backend_data(response.data))
            return {"client": client}
        except Exception as exc:
            return self._backend_error("load_client", exc)

    async def _load_apartment(self, state: OfferState) -> dict:
        try:
            response = await self._apartment_tools.get_apartment(
                state["deal"].apartment_id
            )
            apartment = ApartmentFacts.model_validate(
                require_backend_data(response.data)
            )
            return {"apartment": apartment}
        except Exception as exc:
            return self._backend_error("load_apartment", exc)

    async def _calculate_offer(self, state: OfferState) -> dict:
        try:
            selection = {
                key: value
                for key, value in {
                    "parking_unit_id": state.get("parking_unit_id"),
                    "storage_unit_id": state.get("storage_unit_id"),
                }.items()
                if value is not None
            }
            response = await self._offer_tools.calculate_offer(
                state["deal_id"],
                state["user_id"],
                state["requested_discount_percent"],
                **selection,
            )
            calculation = OfferCalculation.model_validate(
                require_backend_data(response.data)
            )
            if calculation.deal_id != state["deal_id"]:
                raise ValueError("offer calculation deal_id does not match")
        except Exception as exc:
            return self._backend_error("calculate_offer", exc)

        update: dict = {
            "calculation": calculation,
            "approval_required": calculation.requires_approval,
        }
        if not calculation.requires_approval:
            update["approval_status"] = "not_required"
        if state["intent"] == "calculate_offer":
            update["result_message"] = self._format_calculation(
                state["apartment"], calculation
            )
        return update

    async def _generate_text(self, state: OfferState) -> dict:
        context = OfferContext(
            deal=state["deal"],
            client=state["client"],
            apartment=state["apartment"],
            calculation=state["calculation"],
        )
        prompt = (
            "Фактический контекст backend:\n"
            f"{context.model_dump_json(indent=2)}\n\n"
            "Запрос менеджера:\n"
            f"{state['message']}"
        )
        try:
            generated_text = await self._llm.generate(
                prompt,
                system_prompt=OFFER_SYSTEM_PROMPT,
            )
            return {"generated_text": generated_text}
        except Exception as exc:
            logger.warning(
                "Offer text generation failed: error_type=%s", type(exc).__name__
            )
            return {
                "error": "OFFER_GENERATION_ERROR",
                "result_message": OFFER_GENERATION_ERROR_RESPONSE,
            }

    async def _create_offer(self, state: OfferState) -> dict:
        manager_response = self._format_created_offer(
            state["generated_text"],
            state["apartment"],
            state["calculation"],
        )
        try:
            selection = {
                key: value
                for key, value in {
                    "parking_unit_id": state.get("parking_unit_id"),
                    "storage_unit_id": state.get("storage_unit_id"),
                }.items()
                if value is not None
            }
            response = await self._offer_tools.create_offer(
                state["deal_id"],
                state["user_id"],
                state["calculation"].discount_percent,
                state["generated_text"],
                **selection,
            )
            created = OfferCreatedData.model_validate(
                require_backend_data(response.data)
            )
            expected_status = "draft" if state["approval_required"] else "approved"
            if created.status != expected_status:
                raise ValueError(
                    f"offer create response status must be {expected_status}"
                )
            result_message = manager_response
            if not state["approval_required"]:
                result_message = f"{result_message}\n\nПредложение сохранено."
            return {
                "offer_id": created.offer_id,
                "result_message": result_message,
            }
        except Exception as exc:
            logger.warning("Offer creation failed: error_type=%s", type(exc).__name__)
            if isinstance(exc, BackendRpcError) and exc.code == "FORBIDDEN":
                raise
            return {
                "error": "OFFER_CREATE_ERROR",
                "result_message": f"{manager_response}\n\n{OFFER_SAVE_ERROR_RESPONSE}",
            }

    async def _request_approval(self, state: OfferState) -> dict:
        try:
            response = await self._offer_tools.request_offer_approval(
                state["offer_id"],
                state["user_id"],
            )
            approval = OfferApprovalData.model_validate(
                require_backend_data(response.data)
            )
            if approval.offer_id != state["offer_id"]:
                raise ValueError("approval response offer_id does not match")
            if approval.status != "pending_approval":
                raise ValueError("approval response has unexpected status")
            return {
                "approval_status": "waiting",
                "result_message": (
                    f"{state['result_message']}\n\n"
                    "Для указанной скидки требуется согласование руководителя отдела "
                    "продаж. Запрос на согласование отправлен."
                ),
            }
        except Exception as exc:
            logger.warning(
                "Offer approval request failed: error_type=%s", type(exc).__name__
            )
            if isinstance(exc, BackendRpcError) and exc.code == "FORBIDDEN":
                raise
            return {
                "error": "APPROVAL_REQUEST_ERROR",
                "result_message": (
                    f"{state['result_message']}\n\n{APPROVAL_REQUEST_ERROR_RESPONSE}"
                ),
            }

    @staticmethod
    def _route_error(state: OfferState) -> Literal["next", "end"]:
        return "end" if state.get("error") else "next"

    @staticmethod
    def _route_after_calculation(state: OfferState) -> Literal["generate", "end"]:
        if state.get("error") or state["intent"] == "calculate_offer":
            return "end"
        return "generate"

    @staticmethod
    def _route_after_create(state: OfferState) -> Literal["approval", "end"]:
        if state.get("error"):
            return "end"
        return "approval" if state.get("approval_required") else "end"

    @staticmethod
    def _backend_error(node: str, exc: Exception) -> dict:
        code = exc.code if isinstance(exc, BackendRpcError) else None
        logger.warning(
            "Offer graph node failed: node=%s error_type=%s code=%s",
            node,
            type(exc).__name__,
            code,
        )
        if code == "FORBIDDEN":
            raise exc
        return {
            "error": "BACKEND_DATA_ERROR",
            "result_message": BACKEND_DATA_ERROR_RESPONSE,
        }

    @staticmethod
    def _format_created_offer(
        generated_text: str,
        apartment: ApartmentFacts,
        calculation: OfferCalculation,
    ) -> str:
        details = OfferGraph._format_calculation(apartment, calculation)
        return f"Коммерческое предложение подготовлено.\n\n{generated_text}\n\n{details}"

    @staticmethod
    def _format_calculation(
        apartment: ApartmentFacts,
        calculation: OfferCalculation,
    ) -> str:
        apartment_details = [f"Квартира: №{apartment.number or '—'}"]
        if apartment.rooms is not None:
            apartment_details.append(f"{apartment.rooms} комнаты")
        if apartment.area is not None:
            apartment_details.append(f"{apartment.area:g} м²")
        return "\n".join(
            [
                ", ".join(apartment_details),
                f"Базовая стоимость: {OfferGraph._format_money(calculation.apartment_price or 0)} ₽",
                *(
                    [f"Парковка {calculation.parking_number or ''}: {OfferGraph._format_money(calculation.parking_price)} ₽"]
                    if calculation.parking_unit_id is not None
                    else []
                ),
                *(
                    [f"Кладовая {calculation.storage_number or ''}: {OfferGraph._format_money(calculation.storage_price)} ₽"]
                    if calculation.storage_unit_id is not None
                    else []
                ),
                *(
                    [f"Стоимость до скидки: {OfferGraph._format_money(calculation.base_price)} ₽"]
                    if calculation.parking_unit_id is not None
                    or calculation.storage_unit_id is not None
                    else []
                ),
                f"Скидка: {calculation.discount_percent:g}%",
                f"Итоговая стоимость: {OfferGraph._format_money(calculation.final_price)} ₽",
            ]
        )

    @staticmethod
    def _format_money(value: int) -> str:
        return f"{value:,}".replace(",", " ")
