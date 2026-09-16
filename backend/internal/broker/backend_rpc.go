package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"backend/internal/domain"
	"backend/internal/repository"
	"backend/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	BackendRPCQueue  = "backend.agent-rpc"
	AppTopicExchange = "app.topic"

	DealGetRoutingKey                 = "backend.deal.get"
	ClientGetRoutingKey               = "backend.client.get"
	ApartmentGetRoutingKey            = "backend.apartment.get"
	DealMessagesGetRoutingKey         = "backend.deal.messages.get"
	ClientUpdatePreferencesRoutingKey = "backend.client.update_preferences"
	BuildingGetRoutingKey             = "backend.building.get"
	CompetitorListRoutingKey          = "backend.competitor.list"
	ConstructionEventsGetRoutingKey   = "backend.construction.events.get"
	DealListByBuildingRoutingKey      = "backend.deal.list_by_building"
	RecommendationCreateRoutingKey    = "backend.recommendation.create"
	OfferCalculateRoutingKey          = "backend.offer.calculate"
	OfferCreateRoutingKey             = "backend.offer.create"
	OfferRequestApprovalRoutingKey    = "backend.offer.request_approval"
	OfferGetRoutingKey                = "backend.offer.get"
)

var supportedBackendRoutingKeys = []string{
	DealGetRoutingKey,
	ClientGetRoutingKey,
	ApartmentGetRoutingKey,
	DealMessagesGetRoutingKey,
	ClientUpdatePreferencesRoutingKey,
	BuildingGetRoutingKey,
	CompetitorListRoutingKey,
	ConstructionEventsGetRoutingKey,
	DealListByBuildingRoutingKey,
	RecommendationCreateRoutingKey,
	OfferCalculateRoutingKey,
	OfferCreateRoutingKey,
	OfferRequestApprovalRoutingKey,
	OfferGetRoutingKey,
}

type rabbitMQChannelProvider interface {
	NewRabbitMQChannel() (*amqp.Channel, error)
}

type dealReader interface {
	GetDealByID(ctx context.Context, id int) (*domain.Deal, error)
}

type clientReader interface {
	GetUser(ctx context.Context, id int) (*domain.User, error)
}

type apartmentReader interface {
	GetApartmentByID(ctx context.Context, id int) (*domain.Apartment, error)
}

type dealMessagesReader interface {
	GetDealMessages(ctx context.Context, dealID int, limit int) ([]domain.DealDialogMessage, error)
}

type clientPreferencesWriter interface {
	UpdateClientPreferences(ctx context.Context, id int, budgetMax *int64, preferences map[string]any) (*domain.ClientPreferences, error)
}

type constructionReader interface {
	GetBuildingByID(ctx context.Context, id int) (*domain.Building, error)
	GetResidentialComplexByID(ctx context.Context, id int) (*domain.ResidentialComplex, error)
	GetProgressByBuildingID(ctx context.Context, buildingID int) ([]*domain.ConstructionProgress, error)
}

type erpReader interface {
	ListEvents(ctx context.Context, buildingID int) ([]*domain.ERPEvent, error)
	ListMaterialStocks(ctx context.Context, buildingID int) ([]*domain.MaterialStock, error)
	ListProductionSchedules(ctx context.Context, buildingID int) ([]*domain.ProductionSchedule, error)
}

type dealsByBuildingReader interface {
	GetDealsByBuildingID(ctx context.Context, buildingID int) ([]*domain.Deal, error)
}

type competitorReader interface {
	ListByDistrict(ctx context.Context, district string) ([]*domain.Competitor, error)
}

type recommendationWriter interface {
	Create(ctx context.Context, requestID string, recommendation *domain.Recommendation) (*domain.Recommendation, error)
}

type offerWorkflow interface {
	Calculate(ctx context.Context, dealID, requestedBy int, discountPercent string, selections ...domain.OfferSelection) (*domain.OfferCalculation, error)
	Create(ctx context.Context, requestID string, dealID, createdBy int, discountPercent, generatedText string, selections ...domain.OfferSelection) (*domain.Offer, error)
	RequestApproval(ctx context.Context, offerID, requestedBy int) (*domain.Offer, error)
	Get(ctx context.Context, offerID int) (*domain.Offer, error)
}

type BackendRPCDispatcher struct {
	deals            dealReader
	clients          clientReader
	apartments       apartmentReader
	messages         dealMessagesReader
	preferenceWriter clientPreferencesWriter
	construction     constructionReader
	dealsByBuilding  dealsByBuildingReader
	competitors      competitorReader
	recommendations  recommendationWriter
	offers           offerWorkflow
	erp              erpReader
}

func NewBackendRPCDispatcher(
	deals dealReader,
	clients clientReader,
	apartments apartmentReader,
	messages dealMessagesReader,
	preferenceWriter clientPreferencesWriter,
	construction constructionReader,
	dealsByBuilding dealsByBuildingReader,
	competitors competitorReader,
	recommendations recommendationWriter,
	offers offerWorkflow,
	erpReaders ...erpReader,
) *BackendRPCDispatcher {
	var erp erpReader
	if len(erpReaders) > 0 {
		erp = erpReaders[0]
	}
	return &BackendRPCDispatcher{
		deals:            deals,
		clients:          clients,
		apartments:       apartments,
		messages:         messages,
		preferenceWriter: preferenceWriter,
		construction:     construction,
		dealsByBuilding:  dealsByBuilding,
		competitors:      competitors,
		recommendations:  recommendations,
		offers:           offers,
		erp:              erp,
	}
}

type dealGetPayload struct {
	DealID int `json:"deal_id"`
}

type clientGetPayload struct {
	ClientID int `json:"client_id"`
}

type apartmentGetPayload struct {
	ApartmentID int `json:"apartment_id"`
}

type dealMessagesGetPayload struct {
	DealID int `json:"deal_id"`
	Limit  int `json:"limit"`
}

type clientUpdatePreferencesPayload struct {
	ClientID    int            `json:"client_id"`
	BudgetMax   *int64         `json:"budget_max,omitempty"`
	Preferences map[string]any `json:"preferences"`
}

type buildingGetPayload struct {
	BuildingID int `json:"building_id"`
}

type competitorListPayload struct {
	District string `json:"district"`
}

type constructionEventsGetPayload struct {
	BuildingID int `json:"building_id"`
}

type dealListByBuildingPayload struct {
	BuildingID int `json:"building_id"`
}

type recommendationCreatePayload struct {
	DealID         int    `json:"deal_id"`
	Kind           string `json:"kind"`
	Recommendation string `json:"recommendation"`
}

type offerCalculatePayload struct {
	DealID          int         `json:"deal_id"`
	RequestedBy     int         `json:"requested_by"`
	DiscountPercent json.Number `json:"discount_percent"`
	ParkingUnitID   *int        `json:"parking_unit_id,omitempty"`
	StorageUnitID   *int        `json:"storage_unit_id,omitempty"`
}

type offerCreatePayload struct {
	DealID          int         `json:"deal_id"`
	CreatedBy       int         `json:"created_by"`
	DiscountPercent json.Number `json:"discount_percent"`
	GeneratedText   string      `json:"generated_text"`
	ParkingUnitID   *int        `json:"parking_unit_id,omitempty"`
	StorageUnitID   *int        `json:"storage_unit_id,omitempty"`
}

type offerRequestApprovalPayload struct {
	OfferID     int `json:"offer_id"`
	RequestedBy int `json:"requested_by"`
}

type offerGetPayload struct {
	OfferID int `json:"offer_id"`
}

type dealRPCData struct {
	ID          int               `json:"id"`
	ClientID    int               `json:"client_id"`
	ApartmentID int               `json:"apartment_id"`
	Stage       domain.DealStatus `json:"stage"`
}

type clientRPCData struct {
	ID          int            `json:"id"`
	FullName    string         `json:"full_name"`
	BudgetMax   *int64         `json:"budget_max,omitempty"`
	Preferences map[string]any `json:"preferences,omitempty"`
}

type dealMessagesRPCData struct {
	Messages []domain.DealDialogMessage `json:"messages"`
}

type clientUpdatePreferencesRPCData struct {
	ClientID    int            `json:"client_id"`
	BudgetMax   *int64         `json:"budget_max,omitempty"`
	Preferences map[string]any `json:"preferences"`
	Updated     bool           `json:"updated"`
}

type apartmentRPCData struct {
	ID         int                    `json:"id"`
	BuildingID int                    `json:"building_id"`
	Number     string                 `json:"number"`
	Floor      int                    `json:"floor"`
	Rooms      int                    `json:"rooms"`
	Area       float64                `json:"area"`
	Price      float64                `json:"price"`
	Status     domain.ApartmentStatus `json:"status"`
}

type buildingRPCData struct {
	ID               int     `json:"id"`
	Name             string  `json:"name,omitempty"`
	District         string  `json:"district"`
	PlannedDelivery  *string `json:"planned_delivery,omitempty"`
	ForecastDelivery *string `json:"forecast_delivery,omitempty"`
	ReadinessPercent *int    `json:"readiness_percent,omitempty"`
}

type competitorListRPCData struct {
	Competitors []*domain.Competitor `json:"competitors"`
}

type constructionEventRPCData struct {
	Type                 string `json:"type"`
	Title                string `json:"title"`
	RiskLevel            string `json:"risk_level"`
	DelayDays            *int   `json:"delay_days,omitempty"`
	CompletionPercentage *int   `json:"completion_percentage"`
}

type constructionEventsRPCData struct {
	Events []constructionEventRPCData `json:"events"`
}

type affectedDealRPCData struct {
	ID     int               `json:"id"`
	Status domain.DealStatus `json:"status"`
}

type dealsByBuildingRPCData struct {
	Deals []affectedDealRPCData `json:"deals"`
}

type recommendationCreatedRPCData struct {
	RecommendationID int  `json:"recommendation_id"`
	Created          bool `json:"created"`
}

type offerCalculationRPCData struct {
	DealID             int     `json:"deal_id"`
	BasePrice          int64   `json:"base_price"`
	ApartmentPrice     int64   `json:"apartment_price"`
	ParkingUnitID      *int    `json:"parking_unit_id,omitempty"`
	ParkingNumber      *string `json:"parking_number,omitempty"`
	ParkingPrice       int64   `json:"parking_price"`
	StorageUnitID      *int    `json:"storage_unit_id,omitempty"`
	StorageNumber      *string `json:"storage_number,omitempty"`
	StoragePrice       int64   `json:"storage_price"`
	DiscountPercent    float64 `json:"discount_percent"`
	DiscountAmount     int64   `json:"discount_amount"`
	FinalPrice         int64   `json:"final_price"`
	MaxAllowedDiscount float64 `json:"max_allowed_discount"`
	RequiresApproval   bool    `json:"requires_approval"`
}

type offerCreatedRPCData struct {
	OfferID int                `json:"offer_id"`
	Status  domain.OfferStatus `json:"status"`
}

type offerApprovalRPCData struct {
	OfferID int                `json:"offer_id"`
	Status  domain.OfferStatus `json:"status"`
}

type offerGetRPCData struct {
	ID               int                `json:"id"`
	DealID           int                `json:"deal_id"`
	CreatedBy        int                `json:"created_by"`
	BasePrice        int64              `json:"base_price"`
	DiscountPercent  float64            `json:"discount_percent"`
	FinalPrice       int64              `json:"final_price"`
	ParkingUnitID    *int               `json:"parking_unit_id,omitempty"`
	ParkingNumber    *string            `json:"parking_number,omitempty"`
	ParkingPrice     int64              `json:"parking_price"`
	StorageUnitID    *int               `json:"storage_unit_id,omitempty"`
	StorageNumber    *string            `json:"storage_number,omitempty"`
	StoragePrice     int64              `json:"storage_price"`
	GeneratedText    string             `json:"generated_text"`
	Status           domain.OfferStatus `json:"status"`
	ApprovalRequired bool               `json:"approval_required"`
	ApprovedBy       *int               `json:"approved_by"`
	ApprovedAt       *time.Time         `json:"approved_at"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

func (d *BackendRPCDispatcher) Dispatch(
	ctx context.Context,
	routingKey string,
	body []byte,
) domain.BackendRPCResponse {
	requestID := extractRequestID(body)
	var request domain.BackendRPCRequest
	if err := decodeStrict(body, &request); err != nil {
		return domain.BackendRPCFailure(requestID, "VALIDATION_ERROR", "Invalid RPC request")
	}
	if strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(request.Action) == "" {
		return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "request_id and action are required")
	}

	expectedAction, ok := actionForRoutingKey(routingKey)
	if !ok || request.Action != expectedAction {
		return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "action does not match routing key")
	}

	switch routingKey {
	case DealGetRoutingKey:
		var payload dealGetPayload
		if err := decodeStrict(request.Payload, &payload); err != nil {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "Invalid RPC payload")
		}
		if payload.DealID <= 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "deal_id must be a positive integer")
		}
		deal, err := d.deals.GetDealByID(ctx, payload.DealID)
		if err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrDealNotFound, "DEAL_NOT_FOUND", "Deal not found")
		}
		return domain.BackendRPCSuccess(request.RequestID, dealRPCData{
			ID:          deal.ID,
			ClientID:    deal.UserID,
			ApartmentID: deal.ApartmentID,
			Stage:       deal.Status,
		})

	case ClientGetRoutingKey:
		var payload clientGetPayload
		if err := decodeStrict(request.Payload, &payload); err != nil {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "Invalid RPC payload")
		}
		if payload.ClientID <= 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "client_id must be a positive integer")
		}
		client, err := d.clients.GetUser(ctx, payload.ClientID)
		if err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrUserNotFound, "CLIENT_NOT_FOUND", "Client not found")
		}
		return domain.BackendRPCSuccess(request.RequestID, clientRPCData{
			ID:          client.ID,
			FullName:    client.Name,
			BudgetMax:   client.BudgetMax,
			Preferences: client.Preferences,
		})

	case ApartmentGetRoutingKey:
		var payload apartmentGetPayload
		if err := decodeStrict(request.Payload, &payload); err != nil {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "Invalid RPC payload")
		}
		if payload.ApartmentID <= 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "apartment_id must be a positive integer")
		}
		apartment, err := d.apartments.GetApartmentByID(ctx, payload.ApartmentID)
		if err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrApartmentNotFound, "APARTMENT_NOT_FOUND", "Apartment not found")
		}
		return domain.BackendRPCSuccess(request.RequestID, apartmentRPCData{
			ID:         apartment.ID,
			BuildingID: apartment.BuildingID,
			Number:     apartment.Number,
			Floor:      apartment.Floor,
			Rooms:      apartment.Rooms,
			Area:       apartment.Area,
			Price:      apartment.Price,
			Status:     apartment.Status,
		})

	case DealMessagesGetRoutingKey:
		var payload dealMessagesGetPayload
		if err := decodeStrict(request.Payload, &payload); err != nil {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "Invalid RPC payload")
		}
		if payload.DealID <= 0 || payload.Limit <= 0 || payload.Limit > 100 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "deal_id must be positive and limit must be between 1 and 100")
		}
		messages, err := d.messages.GetDealMessages(ctx, payload.DealID, payload.Limit)
		if err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrDealNotFound, "DEAL_NOT_FOUND", "Deal not found")
		}
		return domain.BackendRPCSuccess(request.RequestID, dealMessagesRPCData{Messages: messages})

	case ClientUpdatePreferencesRoutingKey:
		var payload clientUpdatePreferencesPayload
		if err := decodeStrict(request.Payload, &payload); err != nil {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "Invalid RPC payload")
		}
		if payload.ClientID <= 0 || payload.Preferences == nil || (payload.BudgetMax == nil && len(payload.Preferences) == 0) {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "client_id and at least one preference value are required")
		}
		if payload.BudgetMax != nil && *payload.BudgetMax < 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "budget_max must be non-negative")
		}
		updated, err := d.preferenceWriter.UpdateClientPreferences(ctx, payload.ClientID, payload.BudgetMax, payload.Preferences)
		if err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrUserNotFound, "CLIENT_NOT_FOUND", "Client not found")
		}
		return domain.BackendRPCSuccess(request.RequestID, clientUpdatePreferencesRPCData{
			ClientID:    updated.ClientID,
			BudgetMax:   updated.BudgetMax,
			Preferences: updated.Preferences,
			Updated:     true,
		})

	case BuildingGetRoutingKey:
		var payload buildingGetPayload
		if err := decodeStrict(request.Payload, &payload); err != nil || payload.BuildingID <= 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "building_id must be a positive integer")
		}
		building, err := d.construction.GetBuildingByID(ctx, payload.BuildingID)
		if err != nil {
			log.Printf(
				"Backend building lookup failed: request_id=%s building_id=%d error=%v",
				request.RequestID,
				payload.BuildingID,
				err,
			)
			return backendReadFailure(request.RequestID, err, repository.ErrBuildingNotFound, "BUILDING_NOT_FOUND", "Building not found")
		}
		if strings.TrimSpace(building.District) == "" {
			return domain.BackendRPCFailure(request.RequestID, "BUILDING_DATA_INCOMPLETE", "Building district is not configured")
		}
		complex, err := d.construction.GetResidentialComplexByID(ctx, building.ResidentialComplexID)
		if err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrComplexNotFound, "BUILDING_DATA_INCOMPLETE", "Residential complex not found")
		}
		var plannedDelivery *string
		if building.PlannedDate != nil {
			value := building.PlannedDate.Format("2006-01-02")
			plannedDelivery = &value
		}
		var forecastDelivery *string
		if building.ForecastDate != nil {
			value := building.ForecastDate.Format("2006-01-02")
			forecastDelivery = &value
		}
		return domain.BackendRPCSuccess(request.RequestID, buildingRPCData{
			ID: payload.BuildingID, Name: complex.Name, District: building.District,
			PlannedDelivery:  plannedDelivery,
			ForecastDelivery: forecastDelivery, ReadinessPercent: building.ReadinessPercent,
		})

	case CompetitorListRoutingKey:
		var payload competitorListPayload
		if err := decodeStrict(request.Payload, &payload); err != nil || strings.TrimSpace(payload.District) == "" {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "district is required")
		}
		items, err := d.competitors.ListByDistrict(ctx, strings.TrimSpace(payload.District))
		if err != nil {
			return backendReadFailure(request.RequestID, err, nil, "", "")
		}
		return domain.BackendRPCSuccess(request.RequestID, competitorListRPCData{Competitors: items})

	case ConstructionEventsGetRoutingKey:
		var payload constructionEventsGetPayload
		if err := decodeStrict(request.Payload, &payload); err != nil || payload.BuildingID <= 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "building_id must be a positive integer")
		}
		if _, err := d.construction.GetBuildingByID(ctx, payload.BuildingID); err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrBuildingNotFound, "BUILDING_NOT_FOUND", "Building not found")
		}
		progress, err := d.construction.GetProgressByBuildingID(ctx, payload.BuildingID)
		if err != nil {
			return backendReadFailure(request.RequestID, err, nil, "", "")
		}
		events := make([]constructionEventRPCData, 0)
		for _, item := range progress {
			if item.RiskLevel == nil || strings.TrimSpace(*item.RiskLevel) == "" {
				continue
			}
			title := strings.TrimSpace(item.DelayReason)
			if title == "" {
				title = string(item.StageName)
			}
			events = append(events, constructionEventRPCData{
				Type: string(item.StageName), Title: title,
				RiskLevel: *item.RiskLevel, DelayDays: item.DelayDays,
				CompletionPercentage: item.CompletionPercentage,
			})
		}
		if d.erp != nil {
			erpEvents, erpErr := d.erp.ListEvents(ctx, payload.BuildingID)
			if erpErr != nil {
				return backendReadFailure(request.RequestID, erpErr, nil, "", "")
			}
			for _, item := range erpEvents {
				events = append(events, constructionEventRPCData{Type: item.Kind, Title: item.Title, RiskLevel: item.Severity, DelayDays: item.DelayDays})
			}
			stocks, stockErr := d.erp.ListMaterialStocks(ctx, payload.BuildingID)
			if stockErr != nil {
				return backendReadFailure(request.RequestID, stockErr, nil, "", "")
			}
			for _, item := range stocks {
				if item.Quantity < item.MinimumQuantity {
					events = append(events, constructionEventRPCData{Type: "material", Title: "Запас ниже минимального: " + item.MaterialName, RiskLevel: "medium"})
				}
			}
			schedules, scheduleErr := d.erp.ListProductionSchedules(ctx, payload.BuildingID)
			if scheduleErr != nil {
				return backendReadFailure(request.RequestID, scheduleErr, nil, "", "")
			}
			for _, item := range schedules {
				if item.Status == "delayed" {
					events = append(events, constructionEventRPCData{Type: "production", Title: "Задержка производства: " + item.ProductName, RiskLevel: "high"})
				}
			}
		}
		return domain.BackendRPCSuccess(request.RequestID, constructionEventsRPCData{Events: events})

	case DealListByBuildingRoutingKey:
		var payload dealListByBuildingPayload
		if err := decodeStrict(request.Payload, &payload); err != nil || payload.BuildingID <= 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "building_id must be a positive integer")
		}
		if _, err := d.construction.GetBuildingByID(ctx, payload.BuildingID); err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrBuildingNotFound, "BUILDING_NOT_FOUND", "Building not found")
		}
		deals, err := d.dealsByBuilding.GetDealsByBuildingID(ctx, payload.BuildingID)
		if err != nil {
			return backendReadFailure(request.RequestID, err, nil, "", "")
		}
		items := make([]affectedDealRPCData, 0, len(deals))
		for _, deal := range deals {
			items = append(items, affectedDealRPCData{ID: deal.ID, Status: deal.Status})
		}
		return domain.BackendRPCSuccess(request.RequestID, dealsByBuildingRPCData{Deals: items})

	case RecommendationCreateRoutingKey:
		var payload recommendationCreatePayload
		if err := decodeStrict(request.Payload, &payload); err != nil || payload.DealID <= 0 || strings.TrimSpace(payload.Kind) == "" || strings.TrimSpace(payload.Recommendation) == "" {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "deal_id, kind and recommendation are required")
		}
		if _, err := d.deals.GetDealByID(ctx, payload.DealID); err != nil {
			return backendReadFailure(request.RequestID, err, repository.ErrDealNotFound, "DEAL_NOT_FOUND", "Deal not found")
		}
		created, err := d.recommendations.Create(ctx, request.RequestID, &domain.Recommendation{
			DealID: payload.DealID, Kind: strings.TrimSpace(payload.Kind),
			Recommendation: strings.TrimSpace(payload.Recommendation),
		})
		if err != nil {
			return backendReadFailure(request.RequestID, err, nil, "", "")
		}
		return domain.BackendRPCSuccess(request.RequestID, recommendationCreatedRPCData{
			RecommendationID: created.ID, Created: true,
		})

	case OfferCalculateRoutingKey:
		var payload offerCalculatePayload
		if err := decodeStrict(request.Payload, &payload); err != nil || payload.DealID <= 0 || payload.RequestedBy <= 0 || payload.DiscountPercent == "" {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "deal_id, requested_by and discount_percent are required")
		}
		calculation, err := d.offers.Calculate(ctx, payload.DealID, payload.RequestedBy, payload.DiscountPercent.String(), domain.OfferSelection{
			ParkingUnitID: payload.ParkingUnitID,
			StorageUnitID: payload.StorageUnitID,
		})
		if err != nil {
			return offerFailure(request.RequestID, err)
		}
		return domain.BackendRPCSuccess(request.RequestID, calculationRPCData(calculation))

	case OfferCreateRoutingKey:
		var payload offerCreatePayload
		if err := decodeStrict(request.Payload, &payload); err != nil || payload.DealID <= 0 || payload.CreatedBy <= 0 || payload.DiscountPercent == "" || strings.TrimSpace(payload.GeneratedText) == "" {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "deal_id, created_by, discount_percent and generated_text are required")
		}
		offer, err := d.offers.Create(ctx, request.RequestID, payload.DealID, payload.CreatedBy, payload.DiscountPercent.String(), payload.GeneratedText, domain.OfferSelection{
			ParkingUnitID: payload.ParkingUnitID,
			StorageUnitID: payload.StorageUnitID,
		})
		if err != nil {
			return offerFailure(request.RequestID, err)
		}
		return domain.BackendRPCSuccess(request.RequestID, offerCreatedRPCData{OfferID: offer.ID, Status: offer.Status})

	case OfferRequestApprovalRoutingKey:
		var payload offerRequestApprovalPayload
		if err := decodeStrict(request.Payload, &payload); err != nil || payload.OfferID <= 0 || payload.RequestedBy <= 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "offer_id and requested_by are required")
		}
		offer, err := d.offers.RequestApproval(ctx, payload.OfferID, payload.RequestedBy)
		if err != nil {
			return offerFailure(request.RequestID, err)
		}
		return domain.BackendRPCSuccess(request.RequestID, offerApprovalRPCData{OfferID: offer.ID, Status: offer.Status})

	case OfferGetRoutingKey:
		var payload offerGetPayload
		if err := decodeStrict(request.Payload, &payload); err != nil || payload.OfferID <= 0 {
			return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "offer_id must be a positive integer")
		}
		offer, err := d.offers.Get(ctx, payload.OfferID)
		if err != nil {
			return offerFailure(request.RequestID, err)
		}
		return domain.BackendRPCSuccess(request.RequestID, offerGetData(offer))
	}

	return domain.BackendRPCFailure(request.RequestID, "VALIDATION_ERROR", "Unsupported routing key")
}

func backendReadFailure(
	requestID string,
	err error,
	notFound error,
	notFoundCode string,
	notFoundMessage string,
) domain.BackendRPCResponse {
	if errors.Is(err, notFound) {
		return domain.BackendRPCFailure(requestID, notFoundCode, notFoundMessage)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return domain.BackendRPCFailure(requestID, "BACKEND_UNAVAILABLE", "Backend unavailable")
	}
	return domain.BackendRPCFailure(requestID, "INTERNAL_ERROR", "Backend operation failed")
}

func offerFailure(requestID string, err error) domain.BackendRPCResponse {
	switch {
	case errors.Is(err, repository.ErrOfferNotFound):
		return domain.BackendRPCFailure(requestID, "OFFER_NOT_FOUND", "Offer not found")
	case errors.Is(err, repository.ErrDealNotFound):
		return domain.BackendRPCFailure(requestID, "DEAL_NOT_FOUND", "Deal not found")
	case errors.Is(err, repository.ErrApartmentNotFound):
		return domain.BackendRPCFailure(requestID, "APARTMENT_NOT_FOUND", "Apartment not found")
	case errors.Is(err, repository.ErrUserNotFound):
		return domain.BackendRPCFailure(requestID, "USER_NOT_FOUND", "User not found")
	case errors.Is(err, service.ErrInvalidDiscount), errors.Is(err, service.ErrInvalidPrice):
		return domain.BackendRPCFailure(requestID, "INVALID_DISCOUNT", "Discount or apartment price is invalid")
	case errors.Is(err, service.ErrInvalidOfferState):
		return domain.BackendRPCFailure(requestID, "INVALID_OFFER_STATE", "Offer state does not allow this operation")
	case errors.Is(err, service.ErrOfferForbidden):
		return domain.BackendRPCFailure(requestID, "FORBIDDEN", "User is not allowed to perform this offer operation")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return domain.BackendRPCFailure(requestID, "BACKEND_UNAVAILABLE", "Backend unavailable")
	default:
		return domain.BackendRPCFailure(requestID, "INTERNAL_ERROR", "Backend operation failed")
	}
}

func calculationRPCData(calculation *domain.OfferCalculation) offerCalculationRPCData {
	discount, _ := strconv.ParseFloat(calculation.DiscountPercent, 64)
	maximum, _ := strconv.ParseFloat(calculation.MaxAllowedDiscount, 64)
	return offerCalculationRPCData{
		DealID: calculation.DealID, BasePrice: calculation.BasePrice,
		ApartmentPrice: calculation.ApartmentPrice,
		ParkingUnitID:  calculation.ParkingUnitID, ParkingNumber: calculation.ParkingNumber,
		ParkingPrice:  calculation.ParkingPrice,
		StorageUnitID: calculation.StorageUnitID, StorageNumber: calculation.StorageNumber,
		StoragePrice:    calculation.StoragePrice,
		DiscountPercent: discount, DiscountAmount: calculation.DiscountAmount,
		FinalPrice: calculation.FinalPrice, MaxAllowedDiscount: maximum,
		RequiresApproval: calculation.RequiresApproval,
	}
}

func offerGetData(offer *domain.Offer) offerGetRPCData {
	discount, _ := strconv.ParseFloat(offer.DiscountPercent, 64)
	return offerGetRPCData{
		ID: offer.ID, DealID: offer.DealID, CreatedBy: offer.CreatedBy,
		BasePrice: offer.BasePrice, DiscountPercent: discount,
		FinalPrice: offer.FinalPrice, GeneratedText: offer.GeneratedText,
		ParkingUnitID: offer.ParkingUnitID, ParkingNumber: offer.ParkingNumber,
		ParkingPrice: offer.ParkingPrice, StorageUnitID: offer.StorageUnitID,
		StorageNumber: offer.StorageNumber, StoragePrice: offer.StoragePrice,
		Status: offer.Status, ApprovalRequired: offer.ApprovalRequired,
		ApprovedBy: offer.ApprovedBy, ApprovedAt: offer.ApprovedAt,
		CreatedAt: offer.CreatedAt, UpdatedAt: offer.UpdatedAt,
	}
}

func actionForRoutingKey(routingKey string) (string, bool) {
	switch routingKey {
	case DealGetRoutingKey:
		return "deal.get", true
	case ClientGetRoutingKey:
		return "client.get", true
	case ApartmentGetRoutingKey:
		return "apartment.get", true
	case DealMessagesGetRoutingKey:
		return "deal.messages.get", true
	case ClientUpdatePreferencesRoutingKey:
		return "client.update_preferences", true
	case BuildingGetRoutingKey:
		return "building.get", true
	case CompetitorListRoutingKey:
		return "competitor.list", true
	case ConstructionEventsGetRoutingKey:
		return "construction.events.get", true
	case DealListByBuildingRoutingKey:
		return "deal.list_by_building", true
	case RecommendationCreateRoutingKey:
		return "recommendation.create", true
	case OfferCalculateRoutingKey:
		return "offer.calculate", true
	case OfferCreateRoutingKey:
		return "offer.create", true
	case OfferRequestApprovalRoutingKey:
		return "offer.request_approval", true
	case OfferGetRoutingKey:
		return "offer.get", true
	default:
		return "", false
	}
}

func extractRequestID(body []byte) string {
	var metadata struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(body, &metadata); err != nil {
		return ""
	}
	return metadata.RequestID
}

func decodeStrict(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

type BackendRPCConsumer struct {
	provider    rabbitMQChannelProvider
	dispatcher  *BackendRPCDispatcher
	channel     *amqp.Channel
	consumerTag string
	cancel      context.CancelFunc
	waitGroup   sync.WaitGroup
}

func NewBackendRPCConsumer(
	provider rabbitMQChannelProvider,
	dispatcher *BackendRPCDispatcher,
) *BackendRPCConsumer {
	return &BackendRPCConsumer{
		provider:    provider,
		dispatcher:  dispatcher,
		consumerTag: "backend-agent-rpc-consumer",
	}
}

func (c *BackendRPCConsumer) Start(parent context.Context) error {
	channel, err := c.provider.NewRabbitMQChannel()
	if err != nil {
		return fmt.Errorf("open backend RPC channel: %w", err)
	}
	c.channel = channel

	if err := channel.ExchangeDeclare(AppTopicExchange, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		return fmt.Errorf("declare backend RPC exchange: %w", err)
	}
	queue, err := channel.QueueDeclare(BackendRPCQueue, true, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		return fmt.Errorf("declare backend RPC queue: %w", err)
	}
	for _, routingKey := range supportedBackendRoutingKeys {
		if err := channel.QueueBind(queue.Name, routingKey, AppTopicExchange, false, nil); err != nil {
			_ = channel.Close()
			return fmt.Errorf("bind backend RPC routing key %s: %w", routingKey, err)
		}
	}
	if err := channel.Qos(10, 0, false); err != nil {
		_ = channel.Close()
		return fmt.Errorf("configure backend RPC QoS: %w", err)
	}

	deliveries, err := channel.Consume(queue.Name, c.consumerTag, false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		return fmt.Errorf("start backend RPC consumer: %w", err)
	}
	ctx, cancel := context.WithCancel(parent)
	c.cancel = cancel
	c.waitGroup.Add(1)
	go func() {
		defer c.waitGroup.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}
				c.handleDelivery(ctx, delivery)
			}
		}
	}()
	log.Printf("Backend RPC consumer started: queue=%s routing_keys=%v", BackendRPCQueue, supportedBackendRoutingKeys)
	return nil
}

func (c *BackendRPCConsumer) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.channel != nil && !c.channel.IsClosed() {
		_ = c.channel.Cancel(c.consumerTag, false)
		_ = c.channel.Close()
	}
	c.waitGroup.Wait()
	return nil
}

func (c *BackendRPCConsumer) handleDelivery(ctx context.Context, delivery amqp.Delivery) {
	if strings.TrimSpace(delivery.ReplyTo) == "" || strings.TrimSpace(delivery.CorrelationId) == "" {
		log.Printf("Rejecting backend RPC request without reply_to/correlation_id: routing_key=%s", delivery.RoutingKey)
		_ = delivery.Reject(false)
		return
	}

	response := c.dispatcher.Dispatch(ctx, delivery.RoutingKey, delivery.Body)
	if strings.TrimSpace(response.RequestID) == "" {
		log.Printf("Rejecting backend RPC request without request_id: routing_key=%s", delivery.RoutingKey)
		_ = delivery.Reject(false)
		return
	}
	body, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to encode backend RPC response: routing_key=%s", delivery.RoutingKey)
		_ = delivery.Nack(false, true)
		return
	}

	err = c.channel.PublishWithContext(
		ctx,
		"",
		delivery.ReplyTo,
		false,
		false,
		newRPCResponsePublishing(body, delivery.CorrelationId),
	)
	if err != nil {
		log.Printf("Failed to publish backend RPC response: routing_key=%s request_id=%s", delivery.RoutingKey, response.RequestID)
		_ = delivery.Nack(false, true)
		return
	}

	_ = delivery.Ack(false)
	log.Printf("Backend RPC request completed: routing_key=%s request_id=%s", delivery.RoutingKey, response.RequestID)
}

func newRPCResponsePublishing(body []byte, correlationID string) amqp.Publishing {
	return amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: correlationID,
		Body:          body,
	}
}
