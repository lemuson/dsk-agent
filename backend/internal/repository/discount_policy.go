package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDiscountPolicyNotFound = errors.New("discount policy not found")

type DiscountPolicyRepository interface {
	GetActiveMax(ctx context.Context, apartmentID int, role domain.Role) (string, error)
	ListByBuilding(ctx context.Context, buildingID int) ([]*domain.DiscountPolicyRecord, error)
	Create(ctx context.Context, input domain.CreateDiscountPolicyRequest, actorID int) (*domain.DiscountPolicyRecord, error)
}

func (r *discountPolicyRepository) ListByBuilding(ctx context.Context, buildingID int) ([]*domain.DiscountPolicyRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, building_id, role, max_discount_percent::text, version,
		       valid_from, valid_to, created_by, created_at
		FROM discount_policies WHERE building_id = $1
		ORDER BY role, version DESC
	`, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.DiscountPolicyRecord, 0)
	for rows.Next() {
		item := new(domain.DiscountPolicyRecord)
		if err := rows.Scan(&item.ID, &item.BuildingID, &item.Role, &item.MaxDiscountPercent,
			&item.Version, &item.ValidFrom, &item.ValidTo, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *discountPolicyRepository) Create(ctx context.Context, input domain.CreateDiscountPolicyRequest, actorID int) (*domain.DiscountPolicyRecord, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE discount_policies SET valid_to = $1
		WHERE building_id = $2 AND role = $3 AND valid_to IS NULL AND valid_from < $1
	`, input.ValidFrom, input.BuildingID, input.Role); err != nil {
		return nil, err
	}
	item := new(domain.DiscountPolicyRecord)
	err = tx.QueryRow(ctx, `
		INSERT INTO discount_policies (
			building_id, role, max_discount_percent, version, valid_from, created_by
		) VALUES (
			$1, $2, $3,
			(SELECT COALESCE(MAX(version), 0) + 1 FROM discount_policies WHERE building_id = $1 AND role = $2),
			$4, $5
		)
		RETURNING id, building_id, role, max_discount_percent::text, version,
		          valid_from, valid_to, created_by, created_at
	`, input.BuildingID, input.Role, input.MaxDiscountPercent, input.ValidFrom, actorID).Scan(
		&item.ID, &item.BuildingID, &item.Role, &item.MaxDiscountPercent,
		&item.Version, &item.ValidFrom, &item.ValidTo, &item.CreatedBy, &item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

type discountPolicyRepository struct{ db *pgxpool.Pool }

func NewDiscountPolicyRepository(db *pgxpool.Pool) DiscountPolicyRepository {
	return &discountPolicyRepository{db: db}
}

func (r *discountPolicyRepository) GetActiveMax(ctx context.Context, apartmentID int, role domain.Role) (string, error) {
	var value string
	err := r.db.QueryRow(ctx, `
		SELECT policy.max_discount_percent::text
		FROM apartments apartment
		JOIN discount_policies policy ON policy.building_id = apartment.building_id
		WHERE apartment.id = $1
		  AND policy.role = $2
		  AND policy.valid_from <= CURRENT_TIMESTAMP
		  AND (policy.valid_to IS NULL OR policy.valid_to > CURRENT_TIMESTAMP)
		ORDER BY policy.valid_from DESC, policy.version DESC
		LIMIT 1
	`, apartmentID, role).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrDiscountPolicyNotFound
	}
	return value, err
}
