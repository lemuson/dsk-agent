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

type CompetitorHandler struct {
	repository repository.CompetitorRepository
}

func NewCompetitorHandler(repository repository.CompetitorRepository) *CompetitorHandler {
	return &CompetitorHandler{repository: repository}
}

func (h *CompetitorHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repository.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *CompetitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	item, ok := decodeCompetitor(w, r)
	if !ok {
		return
	}
	created, err := h.repository.Create(r.Context(), item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *CompetitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid competitor id", http.StatusBadRequest)
		return
	}
	item, ok := decodeCompetitor(w, r)
	if !ok {
		return
	}
	item.ID = id
	updated, err := h.repository.Update(r.Context(), item)
	if errors.Is(err, repository.ErrCompetitorNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func decodeCompetitor(w http.ResponseWriter, r *http.Request) (*domain.Competitor, bool) {
	var item domain.Competitor
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return nil, false
	}
	item.ProjectName = strings.TrimSpace(item.ProjectName)
	item.District = strings.TrimSpace(item.District)
	if item.ProjectName == "" || item.District == "" || (item.PricePerSqm != nil && *item.PricePerSqm < 0) {
		http.Error(w, "project_name, district and valid price are required", http.StatusBadRequest)
		return nil, false
	}
	return &item, true
}
