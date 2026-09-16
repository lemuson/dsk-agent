package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"backend/internal/domain"
	"backend/internal/repository"
)

var (
	ErrInvalidDiscount   = errors.New("invalid discount")
	ErrInvalidOfferState = errors.New("invalid offer state")
	ErrOfferForbidden    = errors.New("offer operation forbidden")
	ErrInvalidPrice      = errors.New("invalid apartment price")
	ErrInvalidAncillary  = errors.New("invalid ancillary unit")
)

type DiscountPolicy struct {
	ManagerMaxBasisPoints    int64
	SupervisorMaxBasisPoints int64
}

type offerDealReader interface {
	GetDealByID(ctx context.Context, id int) (*domain.Deal, error)
}

type offerUserReader interface {
	GetUserByID(ctx context.Context, id int) (*domain.User, error)
}

type offerStore interface {
	GetApartmentOfferData(ctx context.Context, apartmentID int) (price int64, buildingID int, err error)
	GetAncillaryUnit(ctx context.Context, id int) (*domain.AncillaryUnit, error)
	Create(ctx context.Context, requestID string, offer *domain.Offer) (*domain.Offer, error)
	GetByID(ctx context.Context, id int) (*domain.Offer, error)
	GetByRequestID(ctx context.Context, requestID string) (*domain.Offer, error)
	List(ctx context.Context, createdBy *int) ([]*domain.Offer, error)
	MarkPendingApproval(ctx context.Context, id int) (*domain.Offer, error)
	Decide(ctx context.Context, id, supervisorID int, approve bool, reason string) (*domain.Offer, error)
}

type OfferService interface {
	Calculate(ctx context.Context, dealID, requestedBy int, discountPercent string, selections ...domain.OfferSelection) (*domain.OfferCalculation, error)
	Create(ctx context.Context, requestID string, dealID, createdBy int, discountPercent, generatedText string, selections ...domain.OfferSelection) (*domain.Offer, error)
	RequestApproval(ctx context.Context, offerID, requestedBy int) (*domain.Offer, error)
	Get(ctx context.Context, offerID int) (*domain.Offer, error)
	GetForActor(ctx context.Context, offerID int, actor *domain.User) (*domain.Offer, error)
	ListForActor(ctx context.Context, actor *domain.User) ([]*domain.Offer, error)
	Decide(ctx context.Context, offerID int, actor *domain.User, approve bool, reason string) (*domain.Offer, error)
}

type offerService struct {
	deals        offerDealReader
	users        offerUserReader
	store        offerStore
	policy       DiscountPolicy
	policyReader discountPolicyReader
}

func NewOfferService(
	deals offerDealReader,
	users offerUserReader,
	store offerStore,
	managerMaxDiscount float64,
	supervisorMaxDiscount float64,
	readers ...discountPolicyReader,
) (OfferService, error) {
	manager, err := percentStringToBasisPoints(strconv.FormatFloat(managerMaxDiscount, 'f', -1, 64))
	if err != nil {
		return nil, err
	}
	supervisor, err := percentStringToBasisPoints(strconv.FormatFloat(supervisorMaxDiscount, 'f', -1, 64))
	if err != nil {
		return nil, err
	}
	if supervisor < manager {
		return nil, errors.New("supervisor discount limit must not be lower than manager limit")
	}
	var reader discountPolicyReader
	if len(readers) > 0 {
		reader = readers[0]
	}
	return &offerService{
		deals:        deals,
		users:        users,
		store:        store,
		policy:       DiscountPolicy{ManagerMaxBasisPoints: manager, SupervisorMaxBasisPoints: supervisor},
		policyReader: reader,
	}, nil
}

func (s *offerService) Calculate(
	ctx context.Context,
	dealID int,
	actorID int,
	discountPercent string,
	selections ...domain.OfferSelection,
) (*domain.OfferCalculation, error) {
	selection := domain.OfferSelection{}
	if len(selections) > 0 {
		selection = selections[0]
	}
	discount, err := percentStringToBasisPoints(discountPercent)
	if err != nil {
		return nil, ErrInvalidDiscount
	}
	deal, user, err := s.authorizeDeal(ctx, dealID, actorID)
	if err != nil {
		return nil, err
	}
	if deal.ApartmentID <= 0 {
		return nil, repository.ErrApartmentNotFound
	}
	maxDiscount, err := s.maxDiscount(ctx, deal.ApartmentID, user.Role)
	if err != nil {
		return nil, err
	}
	apartmentPrice, buildingID, err := s.store.GetApartmentOfferData(ctx, deal.ApartmentID)
	if err != nil {
		return nil, err
	}
	if apartmentPrice <= 0 {
		return nil, ErrInvalidPrice
	}
	parking, err := s.ancillarySelection(ctx, selection.ParkingUnitID, buildingID, domain.AncillaryKindParking)
	if err != nil {
		return nil, err
	}
	storage, err := s.ancillarySelection(ctx, selection.StorageUnitID, buildingID, domain.AncillaryKindStorage)
	if err != nil {
		return nil, err
	}
	basePrice := apartmentPrice
	for _, unit := range []*domain.AncillaryUnit{parking, storage} {
		if unit == nil {
			continue
		}
		if unit.Price > math.MaxInt64-basePrice {
			return nil, ErrInvalidPrice
		}
		basePrice += unit.Price
	}
	discountAmount := roundedRatio(basePrice, discount, 10000)
	calculation := &domain.OfferCalculation{
		DealID:             deal.ID,
		BasePrice:          basePrice,
		ApartmentPrice:     apartmentPrice,
		DiscountPercent:    formatBasisPoints(discount),
		DiscountAmount:     discountAmount,
		FinalPrice:         basePrice - discountAmount,
		MaxAllowedDiscount: formatBasisPoints(maxDiscount),
		RequiresApproval:   discount > maxDiscount,
	}
	if parking != nil {
		calculation.ParkingUnitID = &parking.ID
		calculation.ParkingNumber = &parking.Number
		calculation.ParkingPrice = parking.Price
	}
	if storage != nil {
		calculation.StorageUnitID = &storage.ID
		calculation.StorageNumber = &storage.Number
		calculation.StoragePrice = storage.Price
	}
	return calculation, nil
}

func (s *offerService) ancillarySelection(
	ctx context.Context,
	id *int,
	buildingID int,
	kind domain.AncillaryKind,
) (*domain.AncillaryUnit, error) {
	if id == nil {
		return nil, nil
	}
	if *id <= 0 {
		return nil, ErrInvalidAncillary
	}
	unit, err := s.store.GetAncillaryUnit(ctx, *id)
	if err != nil {
		return nil, err
	}
	if unit.BuildingID != buildingID || unit.Kind != kind || unit.Status != domain.ApartmentStatusFree {
		return nil, ErrInvalidAncillary
	}
	return unit, nil
}

func (s *offerService) Create(
	ctx context.Context,
	requestID string,
	dealID int,
	actorID int,
	discountPercent string,
	generatedText string,
	selections ...domain.OfferSelection,
) (*domain.Offer, error) {
	selection := domain.OfferSelection{}
	if len(selections) > 0 {
		selection = selections[0]
	}

	calculation, err := s.Calculate(ctx, dealID, actorID, discountPercent, selection)
	if err != nil {
		return nil, err
	}
	existing, err := s.store.GetByRequestID(ctx, requestID)
	if err == nil {
		if existing.DealID != dealID || existing.CreatedBy != actorID ||
			!sameOptionalID(existing.ParkingUnitID, selection.ParkingUnitID) ||
			!sameOptionalID(existing.StorageUnitID, selection.StorageUnitID) {
			return nil, ErrOfferForbidden
		}
		return existing, nil
	}
	if !errors.Is(err, repository.ErrOfferNotFound) {
		return nil, err
	}
	status := domain.OfferStatusApproved
	if calculation.RequiresApproval {
		status = domain.OfferStatusDraft
	}
	return s.store.Create(ctx, requestID, &domain.Offer{
		DealID:           dealID,
		CreatedBy:        actorID,
		BasePrice:        calculation.BasePrice,
		DiscountPercent:  calculation.DiscountPercent,
		FinalPrice:       calculation.FinalPrice,
		ParkingUnitID:    calculation.ParkingUnitID,
		ParkingNumber:    calculation.ParkingNumber,
		ParkingPrice:     calculation.ParkingPrice,
		StorageUnitID:    calculation.StorageUnitID,
		StorageNumber:    calculation.StorageNumber,
		StoragePrice:     calculation.StoragePrice,
		GeneratedText:    strings.TrimSpace(generatedText),
		Status:           status,
		ApprovalRequired: calculation.RequiresApproval,
	})
}

func sameOptionalID(left, right *int) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func (s *offerService) RequestApproval(ctx context.Context, offerID, actorID int) (*domain.Offer, error) {
	user, err := s.users.GetUserByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	offer, err := s.store.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	deal, err := s.deals.GetDealByID(ctx, offer.DealID)
	if err != nil {
		return nil, err
	}
	if err := authorizeOfferActor(user, deal); err != nil {
		return nil, ErrOfferForbidden
	}
	if !offer.ApprovalRequired {
		return nil, ErrInvalidOfferState
	}
	if offer.Status == domain.OfferStatusPendingApproval {
		return offer, nil
	}
	if offer.Status != domain.OfferStatusDraft {
		return nil, ErrInvalidOfferState
	}
	updated, err := s.store.MarkPendingApproval(ctx, offerID)
	if errors.Is(err, repository.ErrOfferNotFound) {
		current, getErr := s.store.GetByID(ctx, offerID)
		if getErr == nil && current.Status == domain.OfferStatusPendingApproval {
			return current, nil
		}
	}
	return updated, err
}

func (s *offerService) authorizeDeal(
	ctx context.Context,
	dealID int,
	actorID int,
) (*domain.Deal, *domain.User, error) {
	user, err := s.users.GetUserByID(ctx, actorID)
	if err != nil {
		return nil, nil, err
	}
	deal, err := s.deals.GetDealByID(ctx, dealID)
	if err != nil {
		return nil, nil, err
	}
	if err := authorizeOfferActor(user, deal); err != nil {
		return nil, nil, err
	}
	return deal, user, nil
}

func authorizeOfferActor(user *domain.User, deal *domain.Deal) error {
	if user == nil || deal == nil {
		return ErrOfferForbidden
	}
	switch user.Role {
	case domain.RoleSupervisor:
		return nil
	case domain.RoleManager:
		if deal.EmployeeID == user.ID {
			return nil
		}
	}
	return ErrOfferForbidden
}

func (s *offerService) Get(ctx context.Context, offerID int) (*domain.Offer, error) {
	return s.store.GetByID(ctx, offerID)
}

func (s *offerService) GetForActor(ctx context.Context, offerID int, actor *domain.User) (*domain.Offer, error) {
	offer, err := s.store.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	deal, err := s.deals.GetDealByID(ctx, offer.DealID)
	if err != nil {
		return nil, err
	}
	if !canReadOffer(actor, deal) {
		return nil, ErrOfferForbidden
	}
	return offer, nil
}

func (s *offerService) ListForActor(ctx context.Context, actor *domain.User) ([]*domain.Offer, error) {
	if actor == nil {
		return nil, ErrOfferForbidden
	}
	if actor.Role != domain.RoleSupervisor && actor.Role != domain.RoleManager && actor.Role != domain.RoleUser {
		return nil, ErrOfferForbidden
	}
	items, err := s.store.List(ctx, nil)
	if err != nil || actor.Role == domain.RoleSupervisor {
		return items, err
	}
	visible := make([]*domain.Offer, 0, len(items))
	for _, offer := range items {
		deal, dealErr := s.deals.GetDealByID(ctx, offer.DealID)
		if dealErr != nil {
			return nil, dealErr
		}
		if canReadOffer(actor, deal) {
			visible = append(visible, offer)
		}
	}
	return visible, nil
}

func canReadOffer(user *domain.User, deal *domain.Deal) bool {
	if user == nil || deal == nil {
		return false
	}
	return user.Role == domain.RoleSupervisor ||
		(user.Role == domain.RoleManager && deal.EmployeeID == user.ID) ||
		(user.Role == domain.RoleUser && deal.UserID == user.ID)
}

func (s *offerService) Decide(ctx context.Context, offerID int, actor *domain.User, approve bool, reason string) (*domain.Offer, error) {
	if actor == nil || actor.Role != domain.RoleSupervisor {
		return nil, ErrOfferForbidden
	}
	if !approve && strings.TrimSpace(reason) == "" {
		return nil, ErrInvalidOfferState
	}
	offer, err := s.store.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	if offer.Status != domain.OfferStatusPendingApproval || !offer.ApprovalRequired {
		return nil, ErrInvalidOfferState
	}
	updated, err := s.store.Decide(ctx, offerID, actor.ID, approve, strings.TrimSpace(reason))
	if errors.Is(err, repository.ErrOfferNotFound) {
		return nil, ErrInvalidOfferState
	}
	return updated, err
}

func (s *offerService) maxDiscount(ctx context.Context, apartmentID int, role domain.Role) (int64, error) {
	if s.policyReader != nil {
		value, err := s.policyReader.GetActiveMax(ctx, apartmentID, role)
		if err == nil {
			return percentStringToBasisPoints(value)
		}
		if !errors.Is(err, repository.ErrDiscountPolicyNotFound) {
			return 0, err
		}
	}
	switch role {
	case domain.RoleManager:
		return s.policy.ManagerMaxBasisPoints, nil
	case domain.RoleSupervisor:
		return s.policy.SupervisorMaxBasisPoints, nil
	default:
		return 0, ErrOfferForbidden
	}
}

func percentStringToBasisPoints(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return 0, ErrInvalidDiscount
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts[0]) == 0 {
		return 0, ErrInvalidDiscount
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrInvalidDiscount
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) > 2 {
			return 0, ErrInvalidDiscount
		}
	}
	for len(fraction) < 2 {
		fraction += "0"
	}
	fractionValue := int64(0)
	if fraction != "" {
		fractionValue, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, ErrInvalidDiscount
		}
	}
	basisPoints := whole*100 + fractionValue
	if basisPoints < 0 || basisPoints > 10000 {
		return 0, ErrInvalidDiscount
	}
	return basisPoints, nil
}

func formatBasisPoints(value int64) string {
	whole, fraction := value/100, value%100
	if fraction == 0 {
		return strconv.FormatInt(whole, 10)
	}
	if fraction%10 == 0 {
		return strconv.FormatInt(whole, 10) + "." + strconv.FormatInt(fraction/10, 10)
	}
	return strconv.FormatInt(whole, 10) + "." + fmtTwoDigits(fraction)
}

func fmtTwoDigits(value int64) string {
	if value < 10 {
		return "0" + strconv.FormatInt(value, 10)
	}
	return strconv.FormatInt(value, 10)
}

func roundedRatio(value, numerator, denominator int64) int64 {
	return (value*numerator + denominator/2) / denominator
}
