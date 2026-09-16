package domain

import "encoding/json"

type BackendRPCRequest struct {
	RequestID string          `json:"request_id"`
	Action    string          `json:"action"`
	Payload   json.RawMessage `json:"payload"`
}

type BackendRPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BackendRPCResponse struct {
	RequestID string           `json:"request_id"`
	Success   bool             `json:"success"`
	Data      any              `json:"data"`
	Error     *BackendRPCError `json:"error"`
}

func BackendRPCSuccess(requestID string, data any) BackendRPCResponse {
	return BackendRPCResponse{
		RequestID: requestID,
		Success:   true,
		Data:      data,
		Error:     nil,
	}
}

type DealDialogMessage struct {
	ID        int    `json:"id"`
	Direction string `json:"direction"`
	Body      string `json:"body"`
}

type Recommendation struct {
	ID             int    `json:"recommendation_id"`
	DealID         int    `json:"deal_id"`
	Kind           string `json:"kind"`
	Recommendation string `json:"recommendation"`
}

func BackendRPCFailure(requestID, code, message string) BackendRPCResponse {
	return BackendRPCResponse{
		RequestID: requestID,
		Success:   false,
		Data:      nil,
		Error: &BackendRPCError{
			Code:    code,
			Message: message,
		},
	}
}
