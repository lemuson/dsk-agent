package repository

import (
	"context"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ERPRepository interface {
	ListEvents(ctx context.Context, buildingID int) ([]*domain.ERPEvent, error)
	ListMaterialStocks(ctx context.Context, buildingID int) ([]*domain.MaterialStock, error)
	ListProductionSchedules(ctx context.Context, buildingID int) ([]*domain.ProductionSchedule, error)
	CreateEvent(ctx context.Context, item *domain.ERPEvent) error
	UpsertMaterialStock(ctx context.Context, item *domain.MaterialStock) error
	CreateProductionSchedule(ctx context.Context, item *domain.ProductionSchedule) error
}

type erpRepository struct{ db *pgxpool.Pool }

func NewERPRepository(db *pgxpool.Pool) ERPRepository { return &erpRepository{db: db} }

func (r *erpRepository) ListEvents(ctx context.Context, buildingID int) ([]*domain.ERPEvent, error) {
	rows, err := r.db.Query(ctx, `SELECT id, building_id, kind, title, details, severity, affects_delivery, delay_days, occurred_at FROM erp_events WHERE building_id=$1 ORDER BY occurred_at DESC`, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.ERPEvent, 0)
	for rows.Next() {
		item := new(domain.ERPEvent)
		if err := rows.Scan(&item.ID, &item.BuildingID, &item.Kind, &item.Title, &item.Details, &item.Severity, &item.AffectsDelivery, &item.DelayDays, &item.OccurredAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *erpRepository) ListMaterialStocks(ctx context.Context, buildingID int) ([]*domain.MaterialStock, error) {
	rows, err := r.db.Query(ctx, `SELECT id, building_id, material_name, quantity, unit, minimum_quantity, updated_at FROM material_stocks WHERE building_id=$1 ORDER BY material_name`, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.MaterialStock, 0)
	for rows.Next() {
		item := new(domain.MaterialStock)
		if err := rows.Scan(&item.ID, &item.BuildingID, &item.MaterialName, &item.Quantity, &item.Unit, &item.MinimumQuantity, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *erpRepository) ListProductionSchedules(ctx context.Context, buildingID int) ([]*domain.ProductionSchedule, error) {
	rows, err := r.db.Query(ctx, `SELECT id, building_id, product_name, planned_quantity, produced_quantity, planned_date, status, updated_at FROM production_schedules WHERE building_id=$1 ORDER BY planned_date`, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.ProductionSchedule, 0)
	for rows.Next() {
		item := new(domain.ProductionSchedule)
		if err := rows.Scan(&item.ID, &item.BuildingID, &item.ProductName, &item.PlannedQuantity, &item.ProducedQuantity, &item.PlannedDate, &item.Status, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *erpRepository) CreateEvent(ctx context.Context, item *domain.ERPEvent) error {
	return r.db.QueryRow(ctx, `INSERT INTO erp_events (building_id,kind,title,details,severity,affects_delivery,delay_days,occurred_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,occurred_at`, item.BuildingID, item.Kind, item.Title, item.Details, item.Severity, item.AffectsDelivery, item.DelayDays, item.OccurredAt).Scan(&item.ID, &item.OccurredAt)
}
func (r *erpRepository) UpsertMaterialStock(ctx context.Context, item *domain.MaterialStock) error {
	return r.db.QueryRow(ctx, `INSERT INTO material_stocks (building_id,material_name,quantity,unit,minimum_quantity) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (building_id,material_name) DO UPDATE SET quantity=EXCLUDED.quantity,unit=EXCLUDED.unit,minimum_quantity=EXCLUDED.minimum_quantity,updated_at=CURRENT_TIMESTAMP RETURNING id,updated_at`, item.BuildingID, item.MaterialName, item.Quantity, item.Unit, item.MinimumQuantity).Scan(&item.ID, &item.UpdatedAt)
}
func (r *erpRepository) CreateProductionSchedule(ctx context.Context, item *domain.ProductionSchedule) error {
	return r.db.QueryRow(ctx, `INSERT INTO production_schedules (building_id,product_name,planned_quantity,produced_quantity,planned_date,status) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id,updated_at`, item.BuildingID, item.ProductName, item.PlannedQuantity, item.ProducedQuantity, item.PlannedDate, item.Status).Scan(&item.ID, &item.UpdatedAt)
}
