package domain

import "time"

type AncillaryKind string

const (
	AncillaryKindParking AncillaryKind = "parking"
	AncillaryKindStorage AncillaryKind = "storage"
)

type AncillaryUnit struct {
	ID         int             `json:"id"`
	BuildingID int             `json:"building_id"`
	Kind       AncillaryKind   `json:"kind"`
	Number     string          `json:"number"`
	Area       *float64        `json:"area,omitempty"`
	Price      int64           `json:"price"`
	Status     ApartmentStatus `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}
