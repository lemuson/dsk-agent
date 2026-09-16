package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"backend/internal/domain"
	"backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type ERPHandler struct{ repository repository.ERPRepository }

func NewERPHandler(repository repository.ERPRepository) *ERPHandler {
	return &ERPHandler{repository: repository}
}
func erpBuildingID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "buildingId"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid building id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}
func (h *ERPHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := erpBuildingID(w, r)
	if !ok {
		return
	}
	events, err := h.repository.ListEvents(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	stocks, err := h.repository.ListMaterialStocks(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	schedules, err := h.repository.ListProductionSchedules(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, map[string]any{"events": events, "material_stocks": stocks, "production_schedules": schedules})
}
func (h *ERPHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	item := new(domain.ERPEvent)
	if json.NewDecoder(r.Body).Decode(item) != nil || item.BuildingID <= 0 || item.Title == "" {
		http.Error(w, "invalid ERP event", 400)
		return
	}
	if item.OccurredAt.IsZero() {
		item.OccurredAt = time.Now().UTC()
	}
	if err := h.repository.CreateEvent(r.Context(), item); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 201, item)
}
func (h *ERPHandler) UpsertStock(w http.ResponseWriter, r *http.Request) {
	item := new(domain.MaterialStock)
	if json.NewDecoder(r.Body).Decode(item) != nil || item.BuildingID <= 0 || item.MaterialName == "" || item.Quantity < 0 {
		http.Error(w, "invalid material stock", 400)
		return
	}
	if err := h.repository.UpsertMaterialStock(r.Context(), item); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, item)
}
func (h *ERPHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	item := new(domain.ProductionSchedule)
	if json.NewDecoder(r.Body).Decode(item) != nil || item.BuildingID <= 0 || item.ProductName == "" || item.PlannedDate.IsZero() {
		http.Error(w, "invalid production schedule", 400)
		return
	}
	if err := h.repository.CreateProductionSchedule(r.Context(), item); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 201, item)
}
