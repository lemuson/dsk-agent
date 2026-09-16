package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecommendationRepository interface {
	Create(ctx context.Context, requestID string, recommendation *domain.Recommendation) (*domain.Recommendation, error)
}

type recommendationRepository struct {
	db *pgxpool.Pool
}

func NewRecommendationRepository(db *pgxpool.Pool) RecommendationRepository {
	return &recommendationRepository{db: db}
}

func (r *recommendationRepository) Create(
	ctx context.Context,
	requestID string,
	recommendation *domain.Recommendation,
) (*domain.Recommendation, error) {
	query := `
		INSERT INTO recommendations (deal_id, kind, recommendation, request_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (request_id) DO NOTHING
		RETURNING id, deal_id, kind, recommendation
	`
	var result domain.Recommendation
	err := r.db.QueryRow(ctx, query, recommendation.DealID, recommendation.Kind, recommendation.Recommendation, requestID).Scan(
		&result.ID, &result.DealID, &result.Kind, &result.Recommendation,
	)
	if err == nil {
		return &result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	err = r.db.QueryRow(ctx, `SELECT id, deal_id, kind, recommendation FROM recommendations WHERE request_id = $1`, requestID).Scan(
		&result.ID, &result.DealID, &result.Kind, &result.Recommendation,
	)
	return &result, err
}
