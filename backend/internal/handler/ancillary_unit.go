package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/domain"
	"backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type AncillaryUnitHandler struct {
	repository repository.AncillaryUnitRepository
}

func NewAncillaryUnitHandler(repository repository.AncillaryUnitRepository) *AncillaryUnitHandler {
	return &AncillaryUnitHandler{repository: repository}
}

func (h *AncillaryUnitHandler) List(w http.ResponseWriter, r *http.Request) {
	buildingID, err := strconv.Atoi(chi.URLParam(r, "buildingId"))
	if err != nil || buildingID <= 0 {
		http.Error(w, "invalid building id", http.StatusBadRequest)
		return
	}
	items, err := h.repository.ListByBuilding(r.Context(), buildingID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *AncillaryUnitHandler) Create(w http.ResponseWriter, r *http.Request) {
	item := new(domain.AncillaryUnit)
	if err := json.NewDecoder(r.Body).Decode(item); err != nil || !validAncillary(item) {
		http.Error(w, "invalid ancillary unit", http.StatusBadRequest)
		return
	}
	created, err := h.repository.Create(r.Context(), item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *AncillaryUnitHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	item := new(domain.AncillaryUnit)
	if err != nil || json.NewDecoder(r.Body).Decode(item) != nil {
		http.Error(w, "invalid ancillary unit", http.StatusBadRequest)
		return
	}
	item.ID = id
	if !validAncillary(item) {
		http.Error(w, "invalid ancillary unit", http.StatusBadRequest)
		return
	}
	updated, err := h.repository.Update(r.Context(), item)
	if errors.Is(err, repository.ErrAncillaryUnitNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func validAncillary(item *domain.AncillaryUnit) bool {
	return item != nil && item.BuildingID > 0 && strings.TrimSpace(item.Number) != "" && item.Price >= 0 &&
		(item.Kind == domain.AncillaryKindParking || item.Kind == domain.AncillaryKindStorage) &&
		(item.Status == domain.ApartmentStatusFree || item.Status == domain.ApartmentStatusBooked || item.Status == domain.ApartmentStatusSold)
}
