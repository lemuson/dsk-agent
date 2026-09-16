package domain

import "time"

const (
	SenderTypeClient  = "client"
	SenderTypeManager = "manager"
)

type ChatSessionStatus string

const (
	ChatSessionStatusOpen            ChatSessionStatus = "open"
	ChatSessionStatusInProgress      ChatSessionStatus = "in_progress"
	ChatSessionStatusPendingApproval ChatSessionStatus = "pending_approval"
	ChatSessionStatusContract        ChatSessionStatus = "contract"
	ChatSessionStatusClose           ChatSessionStatus = "close"
)

type ChatSession struct {
	ID            int               `json:"id"`
	UserID        *int              `json:"id_user"`
	EmployeeID    *int              `json:"id_employee"`
	ApartmentID   *int              `json:"id_apartment"`
	GuestName     *string           `json:"guest_name,omitempty"`
	GuestEmail    *string           `json:"guest_email,omitempty"`
	GuestPhone    *string           `json:"guest_phone,omitempty"`
	Status        ChatSessionStatus `json:"status"`
	DeletedByUser bool              `json:"deleted_by_user"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	UserName      *string           `json:"user_name,omitempty"`
	EmployeeName  *string           `json:"employee_name,omitempty"`
}

type ChatSessionRejection struct {
	ID            int       `json:"id"`
	ChatSessionID int       `json:"id_chat_sessions"`
	EmployeeID    int       `json:"id_employee"`
	Reason        string    `json:"reason"`
	CreatedAt     time.Time `json:"created_at"`
	EmployeeName  *string   `json:"employee_name,omitempty"`
}

type Message struct {
	ID            int       `json:"id"`
	ChatSessionID int       `json:"id_chat_session"`
	UserID        *int      `json:"id_user"`
	SenderType    string    `json:"sender_type"`
	Content       string    `json:"content"`
	IsRead        bool      `json:"is_read"`
	SendedAt      time.Time `json:"sended_at"`
	SenderName    *string   `json:"sender_name,omitempty"`
}

type CreateChatSessionRequest struct {
	ApartmentID *int   `json:"id_apartment,omitempty"`
	GuestName   string `json:"guest_name,omitempty"`
	GuestEmail  string `json:"guest_email,omitempty"`
	GuestPhone  string `json:"guest_phone,omitempty"`
	Message     string `json:"message,omitempty"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

type RejectSessionRequest struct {
	Reason string `json:"reason"`
}
