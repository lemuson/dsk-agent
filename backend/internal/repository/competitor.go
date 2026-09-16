package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompetitorRepository interface {
	ListByDistrict(ctx context.Context, district string) ([]*domain.Competitor, error)
	List(ctx context.Context) ([]*domain.Competitor, error)
	Create(ctx context.Context, item *domain.Competitor) (*domain.Competitor, error)
	Update(ctx context.Context, item *domain.Competitor) (*domain.Competitor, error)
}

var ErrCompetitorNotFound = errors.New("competitor not found")

type competitorRepository struct {
	db *pgxpool.Pool
}

func NewCompetitorRepository(db *pgxpool.Pool) CompetitorRepository {
	return &competitorRepository{db: db}
}

func (r *competitorRepository) ListByDistrict(ctx context.Context, district string) ([]*domain.Competitor, error) {
	query := `
		SELECT id, project_name, district, price_per_sqm, advantages, disadvantages,
		       source_url, observed_at, updated_at, rooms, area
		FROM competitors
		WHERE LOWER(district) = LOWER($1)
		ORDER BY project_name, id
	`
	rows, err := r.db.Query(ctx, query, district)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.Competitor, 0)
	for rows.Next() {
		var item domain.Competitor
		if err := scanCompetitor(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

func (r *competitorRepository) List(ctx context.Context) ([]*domain.Competitor, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, project_name, district, price_per_sqm, advantages, disadvantages,
		       source_url, observed_at, updated_at, rooms, area
		FROM competitors ORDER BY district, project_name, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.Competitor, 0)
	for rows.Next() {
		var item domain.Competitor
		if err := scanCompetitor(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

func (r *competitorRepository) Create(ctx context.Context, item *domain.Competitor) (*domain.Competitor, error) {
	created := &domain.Competitor{}
	err := scanCompetitor(r.db.QueryRow(ctx, `
		INSERT INTO competitors (project_name, district, price_per_sqm, advantages, disadvantages, source_url, observed_at, rooms, area)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, project_name, district, price_per_sqm, advantages, disadvantages,
		          source_url, observed_at, updated_at, rooms, area
	`, item.ProjectName, item.District, item.PricePerSqm, item.Advantages, item.Disadvantages, item.SourceURL, item.ObservedAt, item.Rooms, item.Area), created)
	return created, err
}

func (r *competitorRepository) Update(ctx context.Context, item *domain.Competitor) (*domain.Competitor, error) {
	updated := &domain.Competitor{}
	err := scanCompetitor(r.db.QueryRow(ctx, `
		UPDATE competitors SET project_name=$1, district=$2, price_per_sqm=$3, advantages=$4,
			disadvantages=$5, source_url=$6, observed_at=$7, rooms=$8, area=$9, updated_at=CURRENT_TIMESTAMP
		WHERE id=$10
		RETURNING id, project_name, district, price_per_sqm, advantages, disadvantages,
		          source_url, observed_at, updated_at, rooms, area
	`, item.ProjectName, item.District, item.PricePerSqm, item.Advantages, item.Disadvantages, item.SourceURL, item.ObservedAt, item.Rooms, item.Area, item.ID), updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCompetitorNotFound
	}
	return updated, err
}

type competitorScanner interface{ Scan(dest ...any) error }

func scanCompetitor(row competitorScanner, item *domain.Competitor) error {
	return row.Scan(&item.ID, &item.ProjectName, &item.District, &item.PricePerSqm,
		&item.Advantages, &item.Disadvantages, &item.SourceURL, &item.ObservedAt,
		&item.UpdatedAt, &item.Rooms, &item.Area)
}
