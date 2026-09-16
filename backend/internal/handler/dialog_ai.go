package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"backend/internal/domain"
	"backend/internal/service"
)

type DialogAIHandler struct {
	agent service.AIAgentService
	deals service.DealService
	audit aiAuditWriter
}

func NewDialogAIHandler(agent service.AIAgentService, deals service.DealService, audits ...aiAuditWriter) *DialogAIHandler {
	var audit aiAuditWriter
	if len(audits) > 0 {
		audit = audits[0]
	}
	return &DialogAIHandler{agent: agent, deals: deals, audit: audit}
}

func (h *DialogAIHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	var input domain.DialogAnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.DealID <= 0 {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	actor := requestUser(r)
	if !h.canUseDeal(r, actor, input.DealID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	result, err := h.agent.AnalyzeDialog(r.Context(), input.DealID, actor.ID)
	if err != nil {
		recordAIAction(r.Context(), h.audit, actor.ID, &input.DealID, nil, "dialog.analyze", "analyze dialog", err.Error(), "analytics", "extract_client_facts", "failed")
		writeAgentError(w, err)
		return
	}
	encoded, _ := json.Marshal(result)
	recordAIAction(r.Context(), h.audit, actor.ID, &input.DealID, nil, "dialog.analyze", "analyze dialog", string(encoded), "analytics", "extract_client_facts", "success")
	writeJSON(w, http.StatusOK, result)
}

func (h *DialogAIHandler) ReplyAssist(w http.ResponseWriter, r *http.Request) {
	var input domain.DialogReplyAssistRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.DealID <= 0 {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if input.SelectedText != nil {
		trimmed := strings.TrimSpace(*input.SelectedText)
		if trimmed == "" {
			input.SelectedText = nil
		} else {
			input.SelectedText = &trimmed
		}
	}
	actor := requestUser(r)
	if !h.canUseDeal(r, actor, input.DealID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	result, err := h.agent.AssistDialogReply(r.Context(), input.DealID, actor.ID, input.SelectedText)
	if err != nil {
		recordAIAction(r.Context(), h.audit, actor.ID, &input.DealID, nil, "dialog.reply_assist", valueOrEmpty(input.SelectedText), err.Error(), "negotiation", "reply_assist", "failed")
		writeAgentError(w, err)
		return
	}
	recordAIAction(r.Context(), h.audit, actor.ID, &input.DealID, nil, "dialog.reply_assist", valueOrEmpty(input.SelectedText), result.SuggestedReply, "negotiation", result.Analysis.Intent, "success")
	writeJSON(w, http.StatusOK, result)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (h *DialogAIHandler) canUseDeal(r *http.Request, actor *domain.User, dealID int) bool {
	if actor == nil {
		return false
	}
	deal, err := h.deals.GetDealByID(r.Context(), dealID)
	if err != nil {
		return false
	}
	return actor.Role == domain.RoleSupervisor || (actor.Role == domain.RoleManager && deal.EmployeeID == actor.ID)
}

func writeAgentError(w http.ResponseWriter, err error) {
	if rpcErr, ok := err.(*service.AgentRPCError); ok {
		status := http.StatusBadGateway
		if rpcErr.Code == "FORBIDDEN" {
			status = http.StatusForbidden
		} else if rpcErr.Code == "NO_CLIENT_MESSAGES" || rpcErr.Code == "VALIDATION_ERROR" {
			status = http.StatusUnprocessableEntity
		}
		http.Error(w, rpcErr.Message, status)
		return
	}
	if err == service.ErrRPCResponseTimeout {
		http.Error(w, err.Error(), http.StatusGatewayTimeout)
		return
	}
	http.Error(w, err.Error(), http.StatusServiceUnavailable)
}
