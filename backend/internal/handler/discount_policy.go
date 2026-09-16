package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/internal/domain"
	"backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

type DiscountPolicyHandler struct{ repository repository.DiscountPolicyRepository }

func NewDiscountPolicyHandler(repository repository.DiscountPolicyRepository) *DiscountPolicyHandler {
	return &DiscountPolicyHandler{repository: repository}
}

func (h *DiscountPolicyHandler) List(w http.ResponseWriter, r *http.Request) {
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

func (h *DiscountPolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, _ := r.Context().Value("user").(*domain.User)
	if actor == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var input domain.CreateDiscountPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.BuildingID <= 0 ||
		(input.Role != domain.RoleManager && input.Role != domain.RoleSupervisor) || input.ValidFrom.IsZero() {
		http.Error(w, "invalid discount policy", http.StatusBadRequest)
		return
	}
	if _, err := strconv.ParseFloat(input.MaxDiscountPercent, 64); err != nil {
		http.Error(w, "invalid discount", http.StatusBadRequest)
		return
	}
	item, err := h.repository.Create(r.Context(), input, actor.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
