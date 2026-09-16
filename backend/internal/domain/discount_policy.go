package domain

import "time"

type DiscountPolicyRecord struct {
	ID                 int        `json:"id"`
	BuildingID         int        `json:"building_id"`
	Role               Role       `json:"role"`
	MaxDiscountPercent string     `json:"max_discount_percent"`
	Version            int        `json:"version"`
	ValidFrom          time.Time  `json:"valid_from"`
	ValidTo            *time.Time `json:"valid_to,omitempty"`
	CreatedBy          int        `json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
}

type CreateDiscountPolicyRequest struct {
	BuildingID         int       `json:"building_id"`
	Role               Role      `json:"role"`
	MaxDiscountPercent string    `json:"max_discount_percent"`
	ValidFrom          time.Time `json:"valid_from"`
}
