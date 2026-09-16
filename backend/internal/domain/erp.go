package domain

import "time"

type ERPEvent struct {
	ID              int       `json:"id"`
	BuildingID      int       `json:"building_id"`
	Kind            string    `json:"kind"`
	Title           string    `json:"title"`
	Details         string    `json:"details"`
	Severity        string    `json:"severity"`
	AffectsDelivery bool      `json:"affects_delivery"`
	DelayDays       *int      `json:"delay_days,omitempty"`
	OccurredAt      time.Time `json:"occurred_at"`
}

type MaterialStock struct {
	ID              int       `json:"id"`
	BuildingID      int       `json:"building_id"`
	MaterialName    string    `json:"material_name"`
	Quantity        float64   `json:"quantity"`
	Unit            string    `json:"unit"`
	MinimumQuantity float64   `json:"minimum_quantity"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ProductionSchedule struct {
	ID               int       `json:"id"`
	BuildingID       int       `json:"building_id"`
	ProductName      string    `json:"product_name"`
	PlannedQuantity  int       `json:"planned_quantity"`
	ProducedQuantity int       `json:"produced_quantity"`
	PlannedDate      time.Time `json:"planned_date"`
	Status           string    `json:"status"`
	UpdatedAt        time.Time `json:"updated_at"`
}
