package domain

type AIChatRequest struct {
	Message       string `json:"message"`
	SessionID     *int   `json:"session_id,omitempty"`
	DealID        *int   `json:"deal_id,omitempty"`
	ParkingUnitID *int   `json:"parking_unit_id,omitempty"`
	StorageUnitID *int   `json:"storage_unit_id,omitempty"`
}

type AgentChatPayload struct {
	UserID        int    `json:"user_id"`
	SessionID     int    `json:"session_id"`
	DealID        *int   `json:"deal_id"`
	ParkingUnitID *int   `json:"parking_unit_id,omitempty"`
	StorageUnitID *int   `json:"storage_unit_id,omitempty"`
	Message       string `json:"message"`
}

type AgentChatRequestEnvelope struct {
	RequestID string           `json:"request_id"`
	Action    string           `json:"action"`
	Payload   AgentChatPayload `json:"payload"`
}

type AgentChatResponseData struct {
	Message string `json:"message"`
	Agent   string `json:"agent"`
	Intent  string `json:"intent"`
}

type AgentChatError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AgentChatResponseEnvelope struct {
	RequestID string                 `json:"request_id"`
	Success   bool                   `json:"success"`
	Data      *AgentChatResponseData `json:"data"`
	Error     *AgentChatError        `json:"error"`
}

type DialogAnalyzeRequest struct {
	DealID int `json:"deal_id"`
}

type DialogReplyAssistRequest struct {
	DealID       int     `json:"deal_id"`
	SelectedText *string `json:"selected_text,omitempty"`
}

type ClientFacts struct {
	BudgetMin          *int64   `json:"budget_min"`
	BudgetMax          *int64   `json:"budget_max"`
	Rooms              *int     `json:"rooms"`
	FloorMin           *int     `json:"floor_min"`
	FloorMax           *int     `json:"floor_max"`
	ParkingRequired    *bool    `json:"parking_required"`
	RenovationRequired *bool    `json:"renovation_required"`
	PreferredDistrict  *string  `json:"preferred_district"`
	PurchaseTimeline   *string  `json:"purchase_timeline"`
	ImportantFactors   []string `json:"important_factors"`
	Objections         []string `json:"objections"`
	Summary            string   `json:"summary"`
}

type DialogAnalyzeResponseData struct {
	DealID             int         `json:"deal_id"`
	ClientID           int         `json:"client_id"`
	Analysis           ClientFacts `json:"analysis"`
	PreferencesUpdated bool        `json:"preferences_updated"`
}

type ReplyAssistAnalysis struct {
	Intent  string `json:"intent"`
	Summary string `json:"summary"`
}

type DialogReplyAssistResponseData struct {
	DealID         int                 `json:"deal_id"`
	Source         string              `json:"source"`
	Analysis       ReplyAssistAnalysis `json:"analysis"`
	SuggestedReply string              `json:"suggested_reply"`
}
