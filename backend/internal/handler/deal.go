package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/domain"
	"backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type DealHandler struct {
	dealService service.DealService
}

func NewDealHandler(dealService service.DealService) *DealHandler {
	return &DealHandler{dealService: dealService}
}

func (h *DealHandler) CreateDeal(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(*domain.User)
	if !ok || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req domain.CreateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	deal, err := h.dealService.CreateDeal(r.Context(), req, user)
	if err != nil {
		if errors.Is(err, service.ErrDealForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrApartmentUnavailable) || errors.Is(err, service.ErrDiscountApprovalRequired) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrInvalidDealData) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(deal)
}

func (h *DealHandler) GetDeal(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid deal id", http.StatusBadRequest)
		return
	}

	deal, err := h.dealService.GetDealByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	user, _ := r.Context().Value("user").(*domain.User)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if user == nil || (user.Role == domain.RoleUser && deal.UserID != user.ID) || (user.Role == domain.RoleManager && deal.EmployeeID != user.ID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(deal)
}

func (h *DealHandler) GetMyDeals(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(*domain.User)
	if !ok || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	deals, err := h.dealService.GetDeals(r.Context(), &user.ID, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(deals)
}

func (h *DealHandler) GetAllDeals(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value("user").(*domain.User)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var statusFilter *domain.DealStatus
	if stParam := r.URL.Query().Get("status"); stParam != "" {
		st := domain.DealStatus(stParam)
		statusFilter = &st
	}

	var userFilter *int
	if uParam := r.URL.Query().Get("user_id"); uParam != "" {
		if id, err := strconv.Atoi(uParam); err == nil {
			userFilter = &id
		}
	}

	var employeeFilter *int
	if user.Role == domain.RoleManager {
		employeeFilter = &user.ID
	} else if user.Role != domain.RoleSupervisor {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	deals, err := h.dealService.GetDeals(r.Context(), userFilter, employeeFilter, statusFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(deals)
}

func (h *DealHandler) UpdateDealStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid deal id", http.StatusBadRequest)
		return
	}

	var req domain.UpdateDealStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, _ := r.Context().Value("user").(*domain.User)
	deal, err := h.dealService.UpdateDealStatus(r.Context(), id, req, user)
	if err != nil {
		if errors.Is(err, service.ErrDealForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrDiscountApprovalRequired) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrInvalidDealData) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(deal)
}
