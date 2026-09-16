package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/domain"
	"backend/internal/repository"
	"backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type OfferHandler struct {
	offers    service.OfferService
	delivery  service.OfferDeliveryService
	documents offerDocumentReader
}

type offerDocumentReader interface {
	GetDocument(ctx context.Context, id int) (*domain.OfferDocument, error)
}

func NewOfferHandler(offers service.OfferService, documents offerDocumentReader, deliveries ...service.OfferDeliveryService) *OfferHandler {
	var delivery service.OfferDeliveryService
	if len(deliveries) > 0 {
		delivery = deliveries[0]
	}
	return &OfferHandler{offers: offers, delivery: delivery, documents: documents}
}

func (h *OfferHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.offers.ListForActor(r.Context(), requestUser(r))
	if err != nil {
		writeOfferError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *OfferHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := offerID(w, r)
	if !ok {
		return
	}
	item, err := h.offers.GetForActor(r.Context(), id, requestUser(r))
	if err != nil {
		writeOfferError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *OfferHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	var input domain.CalculateOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	actor := requestUser(r)
	result, err := h.offers.Calculate(r.Context(), input.DealID, actor.ID, input.DiscountPercent, domain.OfferSelection{
		ParkingUnitID: input.ParkingUnitID,
		StorageUnitID: input.StorageUnitID,
	})
	if err != nil {
		writeOfferError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *OfferHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.RequestID) == "" || strings.TrimSpace(input.GeneratedText) == "" {
		http.Error(w, "request_id and generated_text are required", http.StatusBadRequest)
		return
	}
	actor := requestUser(r)
	result, err := h.offers.Create(r.Context(), input.RequestID, input.DealID, actor.ID, input.DiscountPercent, input.GeneratedText, domain.OfferSelection{
		ParkingUnitID: input.ParkingUnitID,
		StorageUnitID: input.StorageUnitID,
	})
	if err != nil {
		writeOfferError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *OfferHandler) RequestApproval(w http.ResponseWriter, r *http.Request) {
	id, ok := offerID(w, r)
	if !ok {
		return
	}
	result, err := h.offers.RequestApproval(r.Context(), id, requestUser(r).ID)
	if err != nil {
		writeOfferError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *OfferHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, true, "")
}

func (h *OfferHandler) Reject(w http.ResponseWriter, r *http.Request) {
	var input domain.RejectOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	h.decide(w, r, false, input.Reason)
}

func (h *OfferHandler) decide(w http.ResponseWriter, r *http.Request, approve bool, reason string) {
	id, ok := offerID(w, r)
	if !ok {
		return
	}
	result, err := h.offers.Decide(r.Context(), id, requestUser(r), approve, reason)
	if err != nil {
		writeOfferError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *OfferHandler) PDF(w http.ResponseWriter, r *http.Request) {
	id, ok := offerID(w, r)
	if !ok {
		return
	}
	offer, err := h.offers.GetForActor(r.Context(), id, requestUser(r))
	if err != nil {
		writeOfferError(w, err)
		return
	}
	if offer.Status != domain.OfferStatusApproved {
		http.Error(w, "PDF is available only for an approved offer", http.StatusConflict)
		return
	}
	documentData, err := h.documents.GetDocument(r.Context(), offer.ID)
	if err != nil {
		writeOfferError(w, err)
		return
	}
	document, err := buildOfferPDF(documentData)
	if err != nil {
		http.Error(w, "failed to generate PDF", http.StatusInternalServerError)
		return
	}
	if h.delivery != nil {
		if err := h.delivery.Store(r.Context(), offer.ID, document); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="dsk-offer-%d.pdf"`, offer.ID))
	w.Header().Set("Content-Length", strconv.Itoa(len(document)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document)
}

func (h *OfferHandler) Send(w http.ResponseWriter, r *http.Request) {
	if h.delivery == nil {
		http.Error(w, "offer delivery is unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := offerID(w, r)
	if !ok {
		return
	}
	offer, err := h.offers.GetForActor(r.Context(), id, requestUser(r))
	if err != nil {
		writeOfferError(w, err)
		return
	}
	documentData, err := h.documents.GetDocument(r.Context(), offer.ID)
	if err != nil {
		writeOfferError(w, err)
		return
	}
	document, err := buildOfferPDF(documentData)
	if err != nil {
		http.Error(w, "failed to generate PDF", http.StatusInternalServerError)
		return
	}
	result, err := h.delivery.Send(r.Context(), offer, document)
	if err != nil {
		if errors.Is(err, service.ErrOfferEmailNotConfigured) {
			writeJSON(w, http.StatusServiceUnavailable, result)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func offerID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid offer id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func writeOfferError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrOfferForbidden):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, repository.ErrOfferNotFound), errors.Is(err, repository.ErrDealNotFound), errors.Is(err, repository.ErrApartmentNotFound), errors.Is(err, repository.ErrAncillaryUnitNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, service.ErrInvalidDiscount), errors.Is(err, service.ErrInvalidOfferState), errors.Is(err, service.ErrInvalidPrice), errors.Is(err, service.ErrInvalidAncillary):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
