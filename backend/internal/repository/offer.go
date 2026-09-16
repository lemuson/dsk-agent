package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrOfferNotFound = errors.New("offer not found")

type OfferRepository interface {
	GetApartmentOfferData(ctx context.Context, apartmentID int) (price int64, buildingID int, err error)
	GetAncillaryUnit(ctx context.Context, id int) (*domain.AncillaryUnit, error)
	Create(ctx context.Context, requestID string, offer *domain.Offer) (*domain.Offer, error)
	GetByID(ctx context.Context, id int) (*domain.Offer, error)
	GetByRequestID(ctx context.Context, requestID string) (*domain.Offer, error)
	List(ctx context.Context, createdBy *int) ([]*domain.Offer, error)
	MarkPendingApproval(ctx context.Context, id int) (*domain.Offer, error)
	Decide(ctx context.Context, id, supervisorID int, approve bool, reason string) (*domain.Offer, error)
	GetDocument(ctx context.Context, id int) (*domain.OfferDocument, error)
}

type offerRepository struct {
	db *pgxpool.Pool
}

func NewOfferRepository(db *pgxpool.Pool) OfferRepository {
	return &offerRepository{db: db}
}

func (r *offerRepository) GetApartmentOfferData(ctx context.Context, apartmentID int) (int64, int, error) {
	var raw string
	var buildingID int
	if err := r.db.QueryRow(ctx, `SELECT price::text, building_id FROM apartments WHERE id = $1`, apartmentID).Scan(&raw, &buildingID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, ErrApartmentNotFound
		}
		return 0, 0, err
	}
	price, err := decimalMoneyToInteger(raw)
	return price, buildingID, err
}

func (r *offerRepository) GetAncillaryUnit(ctx context.Context, id int) (*domain.AncillaryUnit, error) {
	item, err := scanAncillaryUnit(r.db.QueryRow(ctx, `
		SELECT id, building_id, kind, number, area, price, status, created_at, updated_at
		FROM ancillary_units WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAncillaryUnitNotFound
	}
	return item, err
}

func decimalMoneyToInteger(raw string) (int64, error) {
	parts := strings.SplitN(raw, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	if len(parts) == 1 || strings.TrimRight(parts[1], "0") == "" {
		return whole, nil
	}
	fraction := parts[1]
	if len(fraction) > 2 {
		fraction = fraction[:2]
	}
	for len(fraction) < 2 {
		fraction += "0"
	}
	kopecks, err := strconv.Atoi(fraction)
	if err != nil {
		return 0, err
	}
	if kopecks >= 50 {
		whole++
	}
	return whole, nil
}

func (r *offerRepository) Create(ctx context.Context, requestID string, offer *domain.Offer) (*domain.Offer, error) {
	query := `
		INSERT INTO offers (
			deal_id, created_by, base_price, discount_percent, final_price,
			generated_text, status, approval_required, request_id, version,
			parking_unit_id, parking_price, storage_unit_id, storage_price
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
		          (SELECT COALESCE(MAX(version), 0) + 1 FROM offers WHERE deal_id = $1),
		          $10, $11, $12, $13)
		ON CONFLICT (request_id) DO NOTHING
		RETURNING id
	`
	var id int
	err := r.db.QueryRow(ctx, query,
		offer.DealID, offer.CreatedBy, offer.BasePrice, offer.DiscountPercent,
		offer.FinalPrice, offer.GeneratedText, offer.Status,
		offer.ApprovalRequired, requestID, offer.ParkingUnitID, offer.ParkingPrice,
		offer.StorageUnitID, offer.StoragePrice,
	).Scan(&id)
	if err == nil {
		return r.GetByID(ctx, id)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return r.GetByRequestID(ctx, requestID)
}

func (r *offerRepository) GetByID(ctx context.Context, id int) (*domain.Offer, error) {
	query := `
		SELECT o.id, o.deal_id, o.version, o.created_by, o.base_price, o.discount_percent::text,
		       o.final_price, o.parking_unit_id, parking.number, o.parking_price,
		       o.storage_unit_id, storage.number, o.storage_price,
		       o.generated_text, o.status, o.approval_required,
		       o.approved_by, o.approved_at, o.rejected_by, o.rejected_at, o.rejection_reason,
		       o.created_at, o.updated_at
		FROM offers o
		LEFT JOIN ancillary_units parking ON parking.id = o.parking_unit_id
		LEFT JOIN ancillary_units storage ON storage.id = o.storage_unit_id
		WHERE o.id = $1
	`
	offer, err := scanOffer(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOfferNotFound
	}
	return offer, err
}

func (r *offerRepository) GetDocument(ctx context.Context, id int) (*domain.OfferDocument, error) {
	offer, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	document := &domain.OfferDocument{Offer: offer}
	err = r.db.QueryRow(ctx, `
		SELECT client.name, client.email, manager.name,
		       apartment.id, apartment.number, apartment.rooms, apartment.floor,
		       apartment.area, apartment.type_finishing,
		       complex.name, complex.address,
		       building.id, building.address, COALESCE(building.district, ''),
		       building.readiness_percent, building.planned_date,
		       building.forecast_date, building.delivery_shift_days,
		       parking.area, storage.area
		FROM offers o
		JOIN deals deal ON deal.id = o.deal_id
		JOIN users client ON client.id = deal.id_user
		JOIN users manager ON manager.id = o.created_by
		JOIN apartments apartment ON apartment.id = deal.id_apartment
		JOIN buildings building ON building.id = apartment.building_id
		JOIN residential_complexes complex ON complex.id = building.residential_complex_id
		LEFT JOIN ancillary_units parking ON parking.id = o.parking_unit_id
		LEFT JOIN ancillary_units storage ON storage.id = o.storage_unit_id
		WHERE o.id = $1
	`, id).Scan(
		&document.ClientName, &document.ClientEmail, &document.ManagerName,
		&document.ApartmentID, &document.ApartmentNumber, &document.ApartmentRooms,
		&document.ApartmentFloor, &document.ApartmentArea, &document.ApartmentFinishing,
		&document.ComplexName, &document.ComplexAddress,
		&document.BuildingID, &document.BuildingAddress, &document.BuildingDistrict,
		&document.ReadinessPercent, &document.PlannedDate, &document.ForecastDate,
		&document.DeliveryShiftDays, &document.ParkingArea, &document.StorageArea,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	return document, nil
}

func (r *offerRepository) GetByRequestID(ctx context.Context, requestID string) (*domain.Offer, error) {
	query := `
		SELECT o.id, o.deal_id, o.version, o.created_by, o.base_price, o.discount_percent::text,
		       o.final_price, o.parking_unit_id, parking.number, o.parking_price,
		       o.storage_unit_id, storage.number, o.storage_price,
		       o.generated_text, o.status, o.approval_required,
		       o.approved_by, o.approved_at, o.rejected_by, o.rejected_at, o.rejection_reason,
		       o.created_at, o.updated_at
		FROM offers o
		LEFT JOIN ancillary_units parking ON parking.id = o.parking_unit_id
		LEFT JOIN ancillary_units storage ON storage.id = o.storage_unit_id
		WHERE o.request_id = $1
	`
	offer, err := scanOffer(r.db.QueryRow(ctx, query, requestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOfferNotFound
	}
	return offer, err
}

func (r *offerRepository) List(ctx context.Context, createdBy *int) ([]*domain.Offer, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.id, o.deal_id, o.version, o.created_by, o.base_price, o.discount_percent::text,
		       o.final_price, o.parking_unit_id, parking.number, o.parking_price,
		       o.storage_unit_id, storage.number, o.storage_price,
		       o.generated_text, o.status, o.approval_required,
		       o.approved_by, o.approved_at, o.rejected_by, o.rejected_at, o.rejection_reason,
		       o.created_at, o.updated_at
		FROM offers o
		LEFT JOIN ancillary_units parking ON parking.id = o.parking_unit_id
		LEFT JOIN ancillary_units storage ON storage.id = o.storage_unit_id
		WHERE ($1::int IS NULL OR o.created_by = $1)
		ORDER BY o.updated_at DESC, o.id DESC
	`, createdBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.Offer, 0)
	for rows.Next() {
		item, err := scanOffer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *offerRepository) MarkPendingApproval(ctx context.Context, id int) (*domain.Offer, error) {
	query := `
		UPDATE offers SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND status = $3 AND approval_required = TRUE
		RETURNING id
	`
	var updatedID int
	err := r.db.QueryRow(ctx, query, domain.OfferStatusPendingApproval, id, domain.OfferStatusDraft).Scan(&updatedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, updatedID)
}

func (r *offerRepository) Decide(ctx context.Context, id, supervisorID int, approve bool, reason string) (*domain.Offer, error) {
	status := domain.OfferStatusRejected
	if approve {
		status = domain.OfferStatusApproved
	}
	var updatedID int
	err := r.db.QueryRow(ctx, `
		UPDATE offers SET
			status = $1,
			approved_by = CASE WHEN $2::boolean THEN $3::int ELSE NULL::int END,
			approved_at = CASE WHEN $2::boolean THEN CURRENT_TIMESTAMP ELSE NULL::timestamp END,
			rejected_by = CASE WHEN $2::boolean THEN NULL::int ELSE $3::int END,
			rejected_at = CASE WHEN $2::boolean THEN NULL::timestamp ELSE CURRENT_TIMESTAMP END,
			rejection_reason = CASE WHEN $2::boolean THEN NULL::varchar ELSE NULLIF(TRIM($4::text), '') END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $5 AND status = $6 AND approval_required = TRUE
		RETURNING id
	`, status, approve, supervisorID, reason, id, domain.OfferStatusPendingApproval).Scan(&updatedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, updatedID)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOffer(row rowScanner) (*domain.Offer, error) {
	var offer domain.Offer
	err := row.Scan(
		&offer.ID, &offer.DealID, &offer.Version, &offer.CreatedBy, &offer.BasePrice,
		&offer.DiscountPercent, &offer.FinalPrice,
		&offer.ParkingUnitID, &offer.ParkingNumber, &offer.ParkingPrice,
		&offer.StorageUnitID, &offer.StorageNumber, &offer.StoragePrice,
		&offer.GeneratedText,
		&offer.Status, &offer.ApprovalRequired, &offer.ApprovedBy,
		&offer.ApprovedAt, &offer.RejectedBy, &offer.RejectedAt,
		&offer.RejectionReason, &offer.CreatedAt, &offer.UpdatedAt,
	)
	return &offer, err
}
