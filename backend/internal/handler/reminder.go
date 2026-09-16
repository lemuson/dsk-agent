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

type ReminderHandler struct{ repository repository.ReminderRepository }

func NewReminderHandler(repository repository.ReminderRepository) *ReminderHandler {
	return &ReminderHandler{repository: repository}
}
func (h *ReminderHandler) List(w http.ResponseWriter, r *http.Request) {
	actor := requestUser(r)
	var assigned *int
	if actor.Role != domain.RoleSupervisor {
		assigned = &actor.ID
	}
	items, err := h.repository.List(r.Context(), assigned)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, items)
}
func (h *ReminderHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor := requestUser(r)
	var input domain.CreateStaffReminderRequest
	if json.NewDecoder(r.Body).Decode(&input) != nil || strings.TrimSpace(input.Title) == "" || input.DueAt.IsZero() {
		http.Error(w, "invalid reminder", 400)
		return
	}
	if actor.Role != domain.RoleSupervisor || input.AssignedTo <= 0 {
		input.AssignedTo = actor.ID
	}
	item, err := h.repository.Create(r.Context(), &domain.StaffReminder{AssignedTo: input.AssignedTo, CreatedBy: actor.ID, DealID: input.DealID, Title: strings.TrimSpace(input.Title), DueAt: input.DueAt})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 201, item)
}
func (h *ReminderHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid reminder id", 400)
		return
	}
	item, err := h.repository.Complete(r.Context(), id, requestUser(r))
	if errors.Is(err, repository.ErrReminderNotFound) {
		http.Error(w, err.Error(), 404)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, item)
}
