package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrComplexNotFound   = errors.New("residential complex not found")
	ErrBuildingNotFound  = errors.New("building not found")
	ErrApartmentNotFound = errors.New("apartment not found")
	ErrProgressNotFound  = errors.New("construction progress not found")
)

type ConstructionRepository interface {

	CreateResidentialComplex(ctx context.Context, complex *domain.ResidentialComplex) error
	GetResidentialComplexByID(ctx context.Context, id int) (*domain.ResidentialComplex, error)
	GetAllResidentialComplexes(ctx context.Context) ([]*domain.ResidentialComplex, error)
	UpdateResidentialComplex(ctx context.Context, complex *domain.ResidentialComplex) error
	DeleteResidentialComplex(ctx context.Context, id int) error

	CreateBuilding(ctx context.Context, building *domain.Building) error
	GetBuildingByID(ctx context.Context, id int) (*domain.Building, error)
	GetBuildingsByComplexID(ctx context.Context, complexID int) ([]*domain.Building, error)
	UpdateBuilding(ctx context.Context, building *domain.Building) error
	DeleteBuilding(ctx context.Context, id int) error

	CreateApartment(ctx context.Context, apartment *domain.Apartment) error
	GetApartmentByID(ctx context.Context, id int) (*domain.Apartment, error)
	GetApartmentsByBuildingID(ctx context.Context, buildingID int) ([]*domain.Apartment, error)
	UpdateApartment(ctx context.Context, apartment *domain.Apartment) error
	DeleteApartment(ctx context.Context, id int) error

	CreateProgress(ctx context.Context, progress *domain.ConstructionProgress) error
	GetProgressByID(ctx context.Context, id int) (*domain.ConstructionProgress, error)
	GetProgressByBuildingID(ctx context.Context, buildingID int) ([]*domain.ConstructionProgress, error)
	UpdateProgress(ctx context.Context, progress *domain.ConstructionProgress) error
	DeleteProgress(ctx context.Context, id int) error
}

type postgresConstructionRepository struct {
	db *pgxpool.Pool
}

func NewConstructionRepository(db *pgxpool.Pool) ConstructionRepository {
	return &postgresConstructionRepository{db: db}
}

func (r *postgresConstructionRepository) CreateResidentialComplex(ctx context.Context, c *domain.ResidentialComplex) error {
	query := `INSERT INTO residential_complexes (name, address, description) VALUES ($1, $2, $3) RETURNING id`
	return r.db.QueryRow(ctx, query, c.Name, c.Address, c.Description).Scan(&c.ID)
}

func (r *postgresConstructionRepository) GetResidentialComplexByID(ctx context.Context, id int) (*domain.ResidentialComplex, error) {
	query := `SELECT id, name, address, description FROM residential_complexes WHERE id = $1`
	var c domain.ResidentialComplex
	err := r.db.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.Address, &c.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrComplexNotFound
	}
	return &c, err
}

func (r *postgresConstructionRepository) GetAllResidentialComplexes(ctx context.Context) ([]*domain.ResidentialComplex, error) {
	query := `SELECT id, name, address, description FROM residential_complexes`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var complexes []*domain.ResidentialComplex
	for rows.Next() {
		var c domain.ResidentialComplex
		if err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.Description); err != nil {
			return nil, err
		}
		complexes = append(complexes, &c)
	}
	return complexes, nil
}

func (r *postgresConstructionRepository) UpdateResidentialComplex(ctx context.Context, c *domain.ResidentialComplex) error {
	query := `UPDATE residential_complexes SET name = $1, address = $2, description = $3 WHERE id = $4`
	cmd, err := r.db.Exec(ctx, query, c.Name, c.Address, c.Description, c.ID)
	if err == nil && cmd.RowsAffected() == 0 {
		return ErrComplexNotFound
	}
	return err
}

func (r *postgresConstructionRepository) DeleteResidentialComplex(ctx context.Context, id int) error {
	query := `DELETE FROM residential_complexes WHERE id = $1`
	cmd, err := r.db.Exec(ctx, query, id)
	if err == nil && cmd.RowsAffected() == 0 {
		return ErrComplexNotFound
	}
	return err
}

func (r *postgresConstructionRepository) CreateBuilding(ctx context.Context, b *domain.Building) error {
	query := `INSERT INTO buildings (residential_complex_id, address, district, latitude, longitude, floors_count, planned_date, actual_date, status, type_wall_material, readiness_percent, forecast_date, delivery_shift_days)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id`
	return r.db.QueryRow(ctx, query, b.ResidentialComplexID, b.Address, b.District, b.Latitude, b.Longitude, b.FloorsCount, b.PlannedDate, b.ActualDate, b.Status, b.TypeWallMaterial, b.ReadinessPercent, b.ForecastDate, b.DeliveryShiftDays).Scan(&b.ID)
}

func (r *postgresConstructionRepository) GetBuildingByID(ctx context.Context, id int) (*domain.Building, error) {
	query := `SELECT id, residential_complex_id, address, COALESCE(district, ''), latitude, longitude, floors_count, planned_date, actual_date, status, type_wall_material, readiness_percent, forecast_date, delivery_shift_days FROM buildings WHERE id = $1`
	b, err := scanBuilding(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBuildingNotFound
	}
	return b, err
}

type buildingRow interface {
	Scan(dest ...any) error
}

func scanBuilding(row buildingRow) (*domain.Building, error) {
	var b domain.Building
	var latitude *float64
	var longitude *float64
	err := row.Scan(
		&b.ID,
		&b.ResidentialComplexID,
		&b.Address,
		&b.District,
		&latitude,
		&longitude,
		&b.FloorsCount,
		&b.PlannedDate,
		&b.ActualDate,
		&b.Status,
		&b.TypeWallMaterial,
		&b.ReadinessPercent,
		&b.ForecastDate,
		&b.DeliveryShiftDays,
	)
	if latitude != nil {
		b.Latitude = *latitude
	}
	if longitude != nil {
		b.Longitude = *longitude
	}
	return &b, err
}

func (r *postgresConstructionRepository) GetBuildingsByComplexID(ctx context.Context, complexID int) ([]*domain.Building, error) {
	query := `SELECT id, residential_complex_id, address, COALESCE(district, ''), latitude, longitude, floors_count, planned_date, actual_date, status, type_wall_material, readiness_percent, forecast_date, delivery_shift_days FROM buildings WHERE residential_complex_id = $1`
	rows, err := r.db.Query(ctx, query, complexID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buildings []*domain.Building
	for rows.Next() {
		var b domain.Building
		if err := rows.Scan(&b.ID, &b.ResidentialComplexID, &b.Address, &b.District, &b.Latitude, &b.Longitude, &b.FloorsCount, &b.PlannedDate, &b.ActualDate, &b.Status, &b.TypeWallMaterial, &b.ReadinessPercent, &b.ForecastDate, &b.DeliveryShiftDays); err != nil {
			return nil, err
		}
		buildings = append(buildings, &b)
	}
	return buildings, nil
}

func (r *postgresConstructionRepository) UpdateBuilding(ctx context.Context, b *domain.Building) error {
	query := `UPDATE buildings SET residential_complex_id = $1, address = $2, district = $3, latitude = $4, longitude = $5, floors_count = $6, planned_date = $7, actual_date = $8, status = $9, type_wall_material = $10, readiness_percent = $11, forecast_date = $12, delivery_shift_days = $13 WHERE id = $14`
	cmd, err := r.db.Exec(ctx, query, b.ResidentialComplexID, b.Address, b.District, b.Latitude, b.Longitude, b.FloorsCount, b.PlannedDate, b.ActualDate, b.Status, b.TypeWallMaterial, b.ReadinessPercent, b.ForecastDate, b.DeliveryShiftDays, b.ID)
	if err == nil && cmd.RowsAffected() == 0 {
		return ErrBuildingNotFound
	}
	return err
}

func (r *postgresConstructionRepository) DeleteBuilding(ctx context.Context, id int) error {
	query := `DELETE FROM buildings WHERE id = $1`
	cmd, err := r.db.Exec(ctx, query, id)
	if err == nil && cmd.RowsAffected() == 0 {
		return ErrBuildingNotFound
	}
	return err
}

func (r *postgresConstructionRepository) CreateApartment(ctx context.Context, a *domain.Apartment) error {
	query := `INSERT INTO apartments (building_id, number, rooms, floor, area, price, type_finishing, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	return r.db.QueryRow(ctx, query, a.BuildingID, a.Number, a.Rooms, a.Floor, a.Area, a.Price, a.TypeFinishing, a.Status).Scan(&a.ID)
}

func (r *postgresConstructionRepository) GetApartmentByID(ctx context.Context, id int) (*domain.Apartment, error) {
	query := `SELECT id, building_id, number, rooms, floor, area, price, type_finishing, status FROM apartments WHERE id = $1`
	var a domain.Apartment
	err := r.db.QueryRow(ctx, query, id).Scan(&a.ID, &a.BuildingID, &a.Number, &a.Rooms, &a.Floor, &a.Area, &a.Price, &a.TypeFinishing, &a.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApartmentNotFound
	}
	return &a, err
}

func (r *postgresConstructionRepository) GetApartmentsByBuildingID(ctx context.Context, buildingID int) ([]*domain.Apartment, error) {
	query := `SELECT id, building_id, number, rooms, floor, area, price, type_finishing, status FROM apartments WHERE building_id = $1`
	rows, err := r.db.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apartments []*domain.Apartment
	for rows.Next() {
		var a domain.Apartment
		if err := rows.Scan(&a.ID, &a.BuildingID, &a.Number, &a.Rooms, &a.Floor, &a.Area, &a.Price, &a.TypeFinishing, &a.Status); err != nil {
			return nil, err
		}
		apartments = append(apartments, &a)
	}
	return apartments, nil
}

func (r *postgresConstructionRepository) UpdateApartment(ctx context.Context, a *domain.Apartment) error {
	query := `UPDATE apartments SET building_id = $1, number = $2, rooms = $3, floor = $4, area = $5, price = $6, type_finishing = $7, status = $8 WHERE id = $9`
	cmd, err := r.db.Exec(ctx, query, a.BuildingID, a.Number, a.Rooms, a.Floor, a.Area, a.Price, a.TypeFinishing, a.Status, a.ID)
	if err == nil && cmd.RowsAffected() == 0 {
		return ErrApartmentNotFound
	}
	return err
}

func (r *postgresConstructionRepository) DeleteApartment(ctx context.Context, id int) error {
	query := `DELETE FROM apartments WHERE id = $1`
	cmd, err := r.db.Exec(ctx, query, id)
	if err == nil && cmd.RowsAffected() == 0 {
		return ErrApartmentNotFound
	}
	return err
}

func (r *postgresConstructionRepository) CreateProgress(ctx context.Context, p *domain.ConstructionProgress) error {
	query := `INSERT INTO construction_progress (building_id, stage_name, planned_start_date, actual_start_date, planned_end_date, actual_end_date, status, completion_percentage, delay_reason, risk_level, delay_days)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`
	return r.db.QueryRow(ctx, query, p.BuildingID, p.StageName, p.PlannedStartDate, p.ActualStartDate, p.PlannedEndDate, p.ActualEndDate, p.Status, p.CompletionPercentage, p.DelayReason, p.RiskLevel, p.DelayDays).Scan(&p.ID)
}

func (r *postgresConstructionRepository) GetProgressByID(ctx context.Context, id int) (*domain.ConstructionProgress, error) {
	query := `SELECT id, building_id, stage_name, planned_start_date, actual_start_date, planned_end_date, actual_end_date, status, completion_percentage, COALESCE(delay_reason, ''), risk_level, delay_days FROM construction_progress WHERE id = $1`
	var p domain.ConstructionProgress
	err := r.db.QueryRow(ctx, query, id).Scan(&p.ID, &p.BuildingID, &p.StageName, &p.PlannedStartDate, &p.ActualStartDate, &p.PlannedEndDate, &p.ActualEndDate, &p.Status, &p.CompletionPercentage, &p.DelayReason, &p.RiskLevel, &p.DelayDays)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProgressNotFound
	}
	return &p, err
}

func (r *postgresConstructionRepository) GetProgressByBuildingID(ctx context.Context, buildingID int) ([]*domain.ConstructionProgress, error) {
	query := `SELECT id, building_id, stage_name, planned_start_date, actual_start_date, planned_end_date, actual_end_date, status, completion_percentage, COALESCE(delay_reason, ''), risk_level, delay_days FROM construction_progress WHERE building_id = $1 ORDER BY COALESCE(planned_start_date, actual_start_date), id`
	rows, err := r.db.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progresses []*domain.ConstructionProgress
	for rows.Next() {
		var p domain.ConstructionProgress
		if err := rows.Scan(&p.ID, &p.BuildingID, &p.StageName, &p.PlannedStartDate, &p.ActualStartDate, &p.PlannedEndDate, &p.ActualEndDate, &p.Status, &p.CompletionPercentage, &p.DelayReason, &p.RiskLevel, &p.DelayDays); err != nil {
			return nil, err
		}
		progresses = append(progresses, &p)
	}
	return progresses, nil
}

func (r *postgresConstructionRepository) UpdateProgress(ctx context.Context, p *domain.ConstructionProgress) error {
	query := `UPDATE construction_progress SET building_id = $1, stage_name = $2, planned_start_date = $3, actual_start_date = $4, planned_end_date = $5, actual_end_date = $6, status = $7, completion_percentage = $8, delay_reason = $9, risk_level = $10, delay_days = $11 WHERE id = $12`
	cmd, err := r.db.Exec(ctx, query, p.BuildingID, p.StageName, p.PlannedStartDate, p.ActualStartDate, p.PlannedEndDate, p.ActualEndDate, p.Status, p.CompletionPercentage, p.DelayReason, p.RiskLevel, p.DelayDays, p.ID)
	if err == nil && cmd.RowsAffected() == 0 {
		return ErrProgressNotFound
	}
	return err
}

func (r *postgresConstructionRepository) DeleteProgress(ctx context.Context, id int) error {
	query := `DELETE FROM construction_progress WHERE id = $1`
	cmd, err := r.db.Exec(ctx, query, id)
	if err == nil && cmd.RowsAffected() == 0 {
		return ErrProgressNotFound
	}
	return err
}
