package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/internal/domain"
	"backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	userService service.UserService
	authService service.AuthService
}

func NewUserHandler(userService service.UserService, authService service.AuthService) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
	}
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	actor, _ := r.Context().Value("user").(*domain.User)
	if actor == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var users []*domain.User
	var err error
	if actor.Role == domain.RoleSupervisor {
		users, err = h.userService.GetAllUsers(r.Context())
	} else {
		users, err = h.userService.GetUsersForEmployee(r.Context(), actor.ID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	actor, _ := r.Context().Value("user").(*domain.User)
	if actor == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if actor.Role == domain.RoleManager {
		allowed, accessErr := h.userService.CanEmployeeAccessUser(r.Context(), actor.ID, id)
		if accessErr != nil {
			http.Error(w, accessErr.Error(), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	user, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value("user").(*domain.User)

	user, err := h.userService.GetUser(r.Context(), userCtx.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value("user").(*domain.User)

	var req domain.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	updatedUser, err := h.userService.UpdateUser(r.Context(), userCtx.ID, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newToken, err := h.authService.GenerateToken(updatedUser)
	if err != nil {
		http.Error(w, "failed to generate new token", http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, newToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": newToken,
		"user":  updatedUser,
	})
}
