package domain

import "time"

type OfferStatus string

const (
	OfferStatusDraft           OfferStatus = "draft"
	OfferStatusPendingApproval OfferStatus = "pending_approval"
	OfferStatusApproved        OfferStatus = "approved"
	OfferStatusRejected        OfferStatus = "rejected"
)

type Offer struct {
	ID               int         `json:"id"`
	DealID           int         `json:"deal_id"`
	Version          int         `json:"version"`
	CreatedBy        int         `json:"created_by"`
	BasePrice        int64       `json:"base_price"`
	DiscountPercent  string      `json:"discount_percent"`
	FinalPrice       int64       `json:"final_price"`
	ParkingUnitID    *int        `json:"parking_unit_id,omitempty"`
	ParkingNumber    *string     `json:"parking_number,omitempty"`
	ParkingPrice     int64       `json:"parking_price"`
	StorageUnitID    *int        `json:"storage_unit_id,omitempty"`
	StorageNumber    *string     `json:"storage_number,omitempty"`
	StoragePrice     int64       `json:"storage_price"`
	GeneratedText    string      `json:"generated_text"`
	Status           OfferStatus `json:"status"`
	ApprovalRequired bool        `json:"approval_required"`
	ApprovedBy       *int        `json:"approved_by"`
	ApprovedAt       *time.Time  `json:"approved_at"`
	RejectedBy       *int        `json:"rejected_by"`
	RejectedAt       *time.Time  `json:"rejected_at"`
	RejectionReason  *string     `json:"rejection_reason"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type CreateOfferRequest struct {
	RequestID       string `json:"request_id"`
	DealID          int    `json:"deal_id"`
	DiscountPercent string `json:"discount_percent"`
	GeneratedText   string `json:"generated_text"`
	ParkingUnitID   *int   `json:"parking_unit_id,omitempty"`
	StorageUnitID   *int   `json:"storage_unit_id,omitempty"`
}

type CalculateOfferRequest struct {
	DealID          int    `json:"deal_id"`
	DiscountPercent string `json:"discount_percent"`
	ParkingUnitID   *int   `json:"parking_unit_id,omitempty"`
	StorageUnitID   *int   `json:"storage_unit_id,omitempty"`
}

type OfferSelection struct {
	ParkingUnitID *int `json:"parking_unit_id,omitempty"`
	StorageUnitID *int `json:"storage_unit_id,omitempty"`
}

type RejectOfferRequest struct {
	Reason string `json:"reason"`
}

type OfferCalculation struct {
	DealID             int     `json:"deal_id"`
	BasePrice          int64   `json:"base_price"`
	ApartmentPrice     int64   `json:"apartment_price"`
	ParkingUnitID      *int    `json:"parking_unit_id,omitempty"`
	ParkingNumber      *string `json:"parking_number,omitempty"`
	ParkingPrice       int64   `json:"parking_price"`
	StorageUnitID      *int    `json:"storage_unit_id,omitempty"`
	StorageNumber      *string `json:"storage_number,omitempty"`
	StoragePrice       int64   `json:"storage_price"`
	DiscountPercent    string  `json:"discount_percent"`
	DiscountAmount     int64   `json:"discount_amount"`
	FinalPrice         int64   `json:"final_price"`
	MaxAllowedDiscount string  `json:"max_allowed_discount"`
	RequiresApproval   bool    `json:"requires_approval"`
}

type OfferDocument struct {
	Offer *Offer

	ClientName  string
	ClientEmail string
	ManagerName string

	ApartmentID        int
	ApartmentNumber    string
	ApartmentRooms     int
	ApartmentFloor     int
	ApartmentArea      float64
	ApartmentFinishing FinishingType

	ComplexName       string
	ComplexAddress    string
	BuildingID        int
	BuildingAddress   string
	BuildingDistrict  string
	ReadinessPercent  *int
	PlannedDate       *time.Time
	ForecastDate      *time.Time
	DeliveryShiftDays *int

	ParkingArea *float64
	StorageArea *float64
}
