package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"backend/internal/domain"
	"backend/internal/service"
)

type AIHandler struct {
	aiService   service.AIAgentService
	chatService service.ChatService
	audit       aiAuditWriter
}

type aiAuditWriter interface {
	Record(ctx context.Context, actorID int, dealID, sessionID *int, action, requestText, responseText, agent, intent, status string) error
}

func NewAIHandler(aiService service.AIAgentService, chatService service.ChatService, audits ...aiAuditWriter) *AIHandler {
	var audit aiAuditWriter
	if len(audits) > 0 {
		audit = audits[0]
	}
	return &AIHandler{
		aiService:   aiService,
		chatService: chatService,
		audit:       audit,
	}
}

func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req domain.AIChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		http.Error(w, "message cannot be empty", http.StatusBadRequest)
		return
	}

	if req.SessionID == nil || *req.SessionID <= 0 {
		http.Error(w, "session_id must be a positive integer", http.StatusBadRequest)
		return
	}

	if req.DealID != nil && *req.DealID <= 0 {
		http.Error(w, "deal_id must be a positive integer or null", http.StatusBadRequest)
		return
	}
	if req.ParkingUnitID != nil && *req.ParkingUnitID <= 0 {
		http.Error(w, "parking_unit_id must be a positive integer or null", http.StatusBadRequest)
		return
	}
	if req.StorageUnitID != nil && *req.StorageUnitID <= 0 {
		http.Error(w, "storage_unit_id must be a positive integer or null", http.StatusBadRequest)
		return
	}

	user, _ := r.Context().Value("user").(*domain.User)
	if user == nil || user.ID <= 0 {
		http.Error(w, "authenticated user is required", http.StatusUnauthorized)
		return
	}
	session, err := h.chatService.GetSessionByID(r.Context(), *req.SessionID)
	if err != nil {
		http.Error(w, "chat session not found", http.StatusNotFound)
		return
	}
	if !canReadSession(user, session) || (user.Role == domain.RoleManager && !canManageSession(user, session)) {
		http.Error(w, "FORBIDDEN", http.StatusForbidden)
		return
	}
	userID := user.ID

	payload := domain.AgentChatPayload{
		UserID:        userID,
		SessionID:     *req.SessionID,
		DealID:        req.DealID,
		ParkingUnitID: req.ParkingUnitID,
		StorageUnitID: req.StorageUnitID,
		Message:       req.Message,
	}

	if user.Role == domain.RoleUser {
		userIDForMessage := userID
		_, _ = h.chatService.SendMessage(r.Context(), *req.SessionID, &userIDForMessage, domain.SenderTypeClient, req.Message)
	}

	respData, err := h.aiService.SendChatRequest(r.Context(), payload)
	if err != nil {
		recordAIAction(r.Context(), h.audit, user.ID, req.DealID, req.SessionID, "chat", req.Message, err.Error(), "", "", "failed")
		var agentError *service.AgentRPCError
		if errors.As(err, &agentError) && agentError.Code == "FORBIDDEN" {
			http.Error(w, "FORBIDDEN", http.StatusForbidden)
			return
		}

		if errors.Is(err, service.ErrRabbitMQUnavailable) {
			http.Error(w, "AI service unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		if errors.Is(err, service.ErrRPCResponseTimeout) {
			http.Error(w, "AI response timeout", http.StatusGatewayTimeout)
			return
		}

		http.Error(w, "AI service error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	responseText, agent, intent := "", "", ""
	if respData != nil {
		responseText, agent, intent = respData.Message, respData.Agent, respData.Intent
	}
	recordAIAction(r.Context(), h.audit, user.ID, req.DealID, req.SessionID, "chat", req.Message, responseText, agent, intent, "success")

	if user.Role == domain.RoleUser && respData != nil && respData.Message != "" {
		_, _ = h.chatService.SendMessage(r.Context(), *req.SessionID, nil, "ai", respData.Message)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(respData)
}

func recordAIAction(ctx context.Context, audit aiAuditWriter, actorID int, dealID, sessionID *int, action, requestText, responseText, agent, intent, status string) {
	if audit != nil {
		_ = audit.Record(ctx, actorID, dealID, sessionID, action, requestText, responseText, agent, intent, status)
	}
}
