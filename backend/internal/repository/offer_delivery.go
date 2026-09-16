package repository

import (
	"context"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OfferDeliveryRepository interface {
	SaveDocument(ctx context.Context, offerID int, content []byte, checksum string) error
	CreateDelivery(ctx context.Context, offerID int, recipient string) (*domain.OfferDelivery, error)
	FinishDelivery(ctx context.Context, id int, sent bool, errorMessage string) (*domain.OfferDelivery, error)
}

type offerDeliveryRepository struct{ db *pgxpool.Pool }

func NewOfferDeliveryRepository(db *pgxpool.Pool) OfferDeliveryRepository {
	return &offerDeliveryRepository{db: db}
}

func (r *offerDeliveryRepository) SaveDocument(ctx context.Context, offerID int, content []byte, checksum string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO offer_documents (offer_id,content,checksum_sha256) VALUES ($1,$2,$3) ON CONFLICT (offer_id) DO UPDATE SET content=EXCLUDED.content,checksum_sha256=EXCLUDED.checksum_sha256,generated_at=CURRENT_TIMESTAMP`, offerID, content, checksum)
	return err
}

func (r *offerDeliveryRepository) CreateDelivery(ctx context.Context, offerID int, recipient string) (*domain.OfferDelivery, error) {
	return scanOfferDelivery(r.db.QueryRow(ctx, `INSERT INTO offer_deliveries (offer_id,recipient,channel,status) VALUES ($1,$2,'email','pending') RETURNING id,offer_id,recipient,channel,status,error_message,created_at,sent_at`, offerID, recipient))
}

func (r *offerDeliveryRepository) FinishDelivery(ctx context.Context, id int, sent bool, errorMessage string) (*domain.OfferDelivery, error) {
	return scanOfferDelivery(r.db.QueryRow(ctx, `UPDATE offer_deliveries SET status=CASE WHEN $1 THEN 'sent' ELSE 'failed' END,error_message=CASE WHEN $1 THEN NULL ELSE $2 END,sent_at=CASE WHEN $1 THEN CURRENT_TIMESTAMP ELSE NULL END WHERE id=$3 RETURNING id,offer_id,recipient,channel,status,error_message,created_at,sent_at`, sent, errorMessage, id))
}

func scanOfferDelivery(row rowScanner) (*domain.OfferDelivery, error) {
	item := new(domain.OfferDelivery)
	err := row.Scan(&item.ID, &item.OfferID, &item.Recipient, &item.Channel, &item.Status, &item.ErrorMessage, &item.CreatedAt, &item.SentAt)
	return item, err
}
