package domain

import (
	"time"
)

type BuildingStatus string

const (
	BuildingStatusDesign       BuildingStatus = "design"
	BuildingStatusConstruction BuildingStatus = "construction"
	BuildingStatusCompleted    BuildingStatus = "completed"
	BuildingStatusSuspended    BuildingStatus = "suspended"
)

type WallMaterial string

const (
	WallMaterialPanel    WallMaterial = "panel"
	WallMaterialMonolith WallMaterial = "monolith"
	WallMaterialBrick    WallMaterial = "brick"
	WallMaterialBlock    WallMaterial = "block"
)

type FinishingType string

const (
	FinishingTypeRough    FinishingType = "rough"
	FinishingTypeWhiteBox FinishingType = "white_box"
	FinishingTypeTurnkey  FinishingType = "turnkey"
)

type ApartmentStatus string

const (
	ApartmentStatusFree   ApartmentStatus = "free"
	ApartmentStatusBooked ApartmentStatus = "booked"
	ApartmentStatusSold   ApartmentStatus = "sold"
)

type ProgressStage string

const (
	ProgressStageExcavation ProgressStage = "excavation"
	ProgressStageFoundation ProgressStage = "foundation"
	ProgressStageFrame      ProgressStage = "frame"
	ProgressStageRoofing    ProgressStage = "roofing"
	ProgressStageFinishing  ProgressStage = "finishing"
)

type ProgressStatus string

const (
	ProgressStatusNotStarted ProgressStatus = "not_started"
	ProgressStatusInProgress ProgressStatus = "in_progress"
	ProgressStatusCompleted  ProgressStatus = "completed"
	ProgressStatusDelayed    ProgressStatus = "delayed"
)

type ResidentialComplex struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Description string `json:"description"`
}

type Building struct {
	ID                   int            `json:"id"`
	ResidentialComplexID int            `json:"residential_complex_id"`
	Address              string         `json:"address"`
	District             string         `json:"district"`
	Latitude             float64        `json:"latitude"`
	Longitude            float64        `json:"longitude"`
	FloorsCount          int            `json:"floors_count"`
	PlannedDate          *time.Time     `json:"planned_date"`
	ActualDate           *time.Time     `json:"actual_date"`
	Status               BuildingStatus `json:"status"`
	TypeWallMaterial     WallMaterial   `json:"type_wall_material"`
	ReadinessPercent     *int           `json:"readiness_percent,omitempty"`
	ForecastDate         *time.Time     `json:"forecast_date,omitempty"`
	DeliveryShiftDays    *int           `json:"delivery_shift_days,omitempty"`
}

type Apartment struct {
	ID            int             `json:"id"`
	BuildingID    int             `json:"building_id"`
	Number        string          `json:"number"`
	Rooms         int             `json:"rooms"`
	Floor         int             `json:"floor"`
	Area          float64         `json:"area"`
	Price         float64         `json:"price"`
	TypeFinishing FinishingType   `json:"type_finishing"`
	Status        ApartmentStatus `json:"status"`
}

type ConstructionProgress struct {
	ID                   int            `json:"id"`
	BuildingID           int            `json:"building_id"`
	StageName            ProgressStage  `json:"stage_name"`
	PlannedStartDate     *time.Time     `json:"planned_start_date"`
	ActualStartDate      *time.Time     `json:"actual_start_date"`
	PlannedEndDate       *time.Time     `json:"planned_end_date"`
	ActualEndDate        *time.Time     `json:"actual_end_date"`
	Status               ProgressStatus `json:"status"`
	CompletionPercentage *int           `json:"completion_percentage"`
	DelayReason          string         `json:"delay_reason"`
	RiskLevel            *string        `json:"risk_level,omitempty"`
	DelayDays            *int           `json:"delay_days,omitempty"`
}

type Competitor struct {
	ID            int        `json:"id"`
	ProjectName   string     `json:"project_name"`
	District      string     `json:"district"`
	PricePerSqm   *int64     `json:"price_per_sqm,omitempty"`
	Advantages    *string    `json:"advantages,omitempty"`
	Disadvantages *string    `json:"disadvantages,omitempty"`
	SourceURL     *string    `json:"source_url,omitempty"`
	ObservedAt    *time.Time `json:"observed_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Rooms         *int       `json:"rooms,omitempty"`
	Area          *float64   `json:"area,omitempty"`
}
