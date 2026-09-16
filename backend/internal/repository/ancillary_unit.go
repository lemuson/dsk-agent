package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAncillaryUnitNotFound = errors.New("ancillary unit not found")

type AncillaryUnitRepository interface {
	ListByBuilding(ctx context.Context, buildingID int) ([]*domain.AncillaryUnit, error)
	Create(ctx context.Context, unit *domain.AncillaryUnit) (*domain.AncillaryUnit, error)
	Update(ctx context.Context, unit *domain.AncillaryUnit) (*domain.AncillaryUnit, error)
}

type ancillaryUnitRepository struct{ db *pgxpool.Pool }

func NewAncillaryUnitRepository(db *pgxpool.Pool) AncillaryUnitRepository {
	return &ancillaryUnitRepository{db: db}
}

func (r *ancillaryUnitRepository) ListByBuilding(ctx context.Context, buildingID int) ([]*domain.AncillaryUnit, error) {
	rows, err := r.db.Query(ctx, `SELECT id, building_id, kind, number, area, price, status, created_at, updated_at FROM ancillary_units WHERE building_id = $1 ORDER BY kind, number`, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.AncillaryUnit, 0)
	for rows.Next() {
		item, err := scanAncillaryUnit(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ancillaryUnitRepository) Create(ctx context.Context, unit *domain.AncillaryUnit) (*domain.AncillaryUnit, error) {
	return scanAncillaryUnit(r.db.QueryRow(ctx, `INSERT INTO ancillary_units (building_id, kind, number, area, price, status) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, building_id, kind, number, area, price, status, created_at, updated_at`, unit.BuildingID, unit.Kind, unit.Number, unit.Area, unit.Price, unit.Status))
}

func (r *ancillaryUnitRepository) Update(ctx context.Context, unit *domain.AncillaryUnit) (*domain.AncillaryUnit, error) {
	item, err := scanAncillaryUnit(r.db.QueryRow(ctx, `UPDATE ancillary_units SET kind=$1, number=$2, area=$3, price=$4, status=$5, updated_at=CURRENT_TIMESTAMP WHERE id=$6 RETURNING id, building_id, kind, number, area, price, status, created_at, updated_at`, unit.Kind, unit.Number, unit.Area, unit.Price, unit.Status, unit.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAncillaryUnitNotFound
	}
	return item, err
}

func scanAncillaryUnit(row rowScanner) (*domain.AncillaryUnit, error) {
	item := new(domain.AncillaryUnit)
	err := row.Scan(&item.ID, &item.BuildingID, &item.Kind, &item.Number, &item.Area, &item.Price, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}
