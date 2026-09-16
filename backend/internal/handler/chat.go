package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/domain"
	"backend/internal/service"
	"github.com/go-chi/chi/v5"
)

type ChatHandler struct {
	chatService service.ChatService
}

func NewChatHandler(chatService service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func requestUser(r *http.Request) *domain.User {
	user, _ := r.Context().Value("user").(*domain.User)
	return user
}

func canReadSession(user *domain.User, session *domain.ChatSession) bool {
	if user == nil || session == nil {
		return false
	}
	switch user.Role {
	case domain.RoleSupervisor:
		return true
	case domain.RoleManager:
		return session.EmployeeID == nil || *session.EmployeeID == user.ID
	case domain.RoleUser:
		return session.UserID != nil && *session.UserID == user.ID
	default:
		return false
	}
}

func canManageSession(user *domain.User, session *domain.ChatSession) bool {
	if user == nil || session == nil {
		return false
	}
	if user.Role == domain.RoleSupervisor {
		return true
	}
	if user.Role == domain.RoleManager {
		if session.ApartmentID == nil {
			return true
		}
		return session.EmployeeID != nil && *session.EmployeeID == user.ID
	}
	return false
}

func (h *ChatHandler) findAuthorizedSession(w http.ResponseWriter, r *http.Request, id int, manage bool) (*domain.ChatSession, bool) {
	session, err := h.chatService.GetSessionByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil, false
	}
	allowed := canReadSession(requestUser(r), session)
	if manage {
		allowed = canManageSession(requestUser(r), session)
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return session, true
}

func (h *ChatHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateChatSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var userID *int
	if user := requestUser(r); user != nil {
		userID = &user.ID
	}
	session, err := h.chatService.CreateSession(r.Context(), req, userID)
	if err != nil {
		if err == service.ErrInvalidSessionData {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(session)
}

func (h *ChatHandler) GetMySessions(w http.ResponseWriter, r *http.Request) {
	user := requestUser(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sessions, err := h.chatService.GetSessions(r.Context(), &user.ID, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessions)
}

func (h *ChatHandler) GetAllSessions(w http.ResponseWriter, r *http.Request) {
	user := requestUser(r)
	var statusFilter *domain.ChatSessionStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		status := domain.ChatSessionStatus(raw)
		statusFilter = &status
	}
	var employeeFilter *int
	if raw := r.URL.Query().Get("employee_id"); raw != "" && user.Role == domain.RoleSupervisor {
		id, err := strconv.Atoi(raw)
		if err != nil {
			http.Error(w, "invalid employee id", http.StatusBadRequest)
			return
		}
		employeeFilter = &id
	}
	sessions, err := h.chatService.GetSessions(r.Context(), nil, employeeFilter, statusFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user.Role == domain.RoleManager {
		visible := make([]*domain.ChatSession, 0, len(sessions))
		for _, session := range sessions {
			if canReadSession(user, session) {
				visible = append(visible, session)
			}
		}
		sessions = visible
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessions)
}

func (h *ChatHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	session, ok := h.findAuthorizedSession(w, r, id, false)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}

func (h *ChatHandler) TakeSession(w http.ResponseWriter, r *http.Request) {
	user := requestUser(r)
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	session, err := h.chatService.GetSessionByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if session.ApartmentID == nil {
		http.Error(w, "general questions do not require taking into work", http.StatusBadRequest)
		return
	}
	if session.EmployeeID != nil {
		http.Error(w, "session is already assigned", http.StatusConflict)
		return
	}
	if err := h.chatService.TakeSession(r.Context(), id, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	session, err = h.chatService.GetSessionByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}

func (h *ChatHandler) CloseSession(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	if _, ok := h.findAuthorizedSession(w, r, id, true); !ok {
		return
	}
	if err := h.chatService.CloseSession(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "session closed successfully"})
}

func (h *ChatHandler) RejectSession(w http.ResponseWriter, r *http.Request) {
	user := requestUser(r)
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	if _, ok := h.findAuthorizedSession(w, r, id, true); !ok {
		return
	}
	var req domain.RejectSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	rejection, err := h.chatService.RejectSession(r.Context(), id, user.ID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(rejection)
}

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	user := requestUser(r)
	session, ok := h.findAuthorizedSession(w, r, id, false)
	if !ok || (user.Role != domain.RoleUser && !canManageSession(user, session)) {
		if ok {
			http.Error(w, "forbidden", http.StatusForbidden)
		}
		return
	}
	var req domain.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	userID := &user.ID
	senderType := domain.SenderTypeClient
	if user.Role == domain.RoleManager || user.Role == domain.RoleSupervisor {
		senderType = domain.SenderTypeManager
	}
	msg, err := h.chatService.SendMessage(r.Context(), id, userID, senderType, req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(msg)
}

func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	if _, ok := h.findAuthorizedSession(w, r, id, false); !ok {
		return
	}
	user := requestUser(r)
	messages, err := h.chatService.GetMessages(r.Context(), id, &user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(messages)
}

func (h *ChatHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	user := requestUser(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.chatService.DeleteSessionForUser(r.Context(), id, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "session hidden successfully"})
}

