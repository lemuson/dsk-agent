package domain

import "time"

type StaffReminder struct {
	ID          int64      `json:"id"`
	AssignedTo  int        `json:"assigned_to"`
	CreatedBy   int        `json:"created_by"`
	DealID      *int       `json:"deal_id,omitempty"`
	Title       string     `json:"title"`
	DueAt       time.Time  `json:"due_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateStaffReminderRequest struct {
	AssignedTo int       `json:"assigned_to"`
	DealID     *int      `json:"deal_id,omitempty"`
	Title      string    `json:"title"`
	DueAt      time.Time `json:"due_at"`
}
