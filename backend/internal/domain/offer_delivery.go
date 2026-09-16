package domain

import "time"

type OfferDelivery struct {
	ID           int        `json:"id"`
	OfferID      int        `json:"offer_id"`
	Recipient    string     `json:"recipient"`
	Channel      string     `json:"channel"`
	Status       string     `json:"status"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
}
