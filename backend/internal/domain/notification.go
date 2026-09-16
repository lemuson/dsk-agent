package domain

import "time"

type NotificationType string

const (
	NotificationTypeConstructionDelay NotificationType = "construction_delay"
	NotificationTypeConstructionRisk  NotificationType = "construction_risk"
	NotificationTypeDealUpdate        NotificationType = "deal_update"
	NotificationTypeGeneral           NotificationType = "general"
)

type Notification struct {
	ID        int              `json:"id"`
	UserID    int              `json:"user_id"`
	DealID    *int             `json:"deal_id,omitempty"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	IsRead    bool             `json:"is_read"`
	CreatedAt time.Time        `json:"created_at"`
	ReadAt    *time.Time       `json:"read_at,omitempty"`
}
