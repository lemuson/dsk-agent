package domain

import "time"

type DealStatus string

const (
	DealStatusPending   DealStatus = "pending"
	DealStatusContract  DealStatus = "contract"
	DealStatusCompleted DealStatus = "completed"
	DealStatusCancelled DealStatus = "cancelled"
)

type Deal struct {
	ID              int        `json:"id"`
	UserID          int        `json:"id_user"`
	EmployeeID      int        `json:"id_employee"`
	ApartmentID     int        `json:"id_apartment"`
	ChatSessionID   *int       `json:"id_chat_session,omitempty"`
	BasePrice       float64    `json:"base_price"`
	PercentDiscount float64    `json:"percent_discount"`
	TotalPrice      float64    `json:"total_price"`
	Status          DealStatus `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	UserName        *string `json:"user_name,omitempty"`
	EmployeeName    *string `json:"employee_name,omitempty"`
	ApartmentNumber *string `json:"apartment_number,omitempty"`
}

type CreateDealRequest struct {
	UserID          int     `json:"id_user"`
	ApartmentID     int     `json:"id_apartment"`
	ChatSessionID   *int    `json:"id_chat_session,omitempty"`
	PercentDiscount float64 `json:"percent_discount"`
}

type UpdateDealStatusRequest struct {
	Status          DealStatus `json:"status"`
	PercentDiscount *float64   `json:"percent_discount,omitempty"`
}
