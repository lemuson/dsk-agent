package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotificationNotFound = errors.New("notification not found")

type NotificationRepository interface {
	CreateNotification(ctx context.Context, n *domain.Notification) (*domain.Notification, error)
	ClaimConstructionState(ctx context.Context, progressID int, fingerprint string) (bool, error)
	GetNotificationsByUserID(ctx context.Context, userID int, unreadOnly bool) ([]*domain.Notification, error)
	MarkAsRead(ctx context.Context, id int, userID int) error
	MarkAllAsRead(ctx context.Context, userID int) error
}

func (r *postgresNotificationRepository) ClaimConstructionState(ctx context.Context, progressID int, fingerprint string) (bool, error) {
	var claimed int
	err := r.db.QueryRow(ctx, `
		INSERT INTO construction_notification_state (progress_id, fingerprint, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (progress_id) DO UPDATE
		SET fingerprint = EXCLUDED.fingerprint, updated_at = NOW()
		WHERE construction_notification_state.fingerprint IS DISTINCT FROM EXCLUDED.fingerprint
		RETURNING progress_id
	`, progressID, fingerprint).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

type postgresNotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) NotificationRepository {
	return &postgresNotificationRepository{db: db}
}

func (r *postgresNotificationRepository) CreateNotification(ctx context.Context, n *domain.Notification) (*domain.Notification, error) {
	query := `
		INSERT INTO notifications (user_id, deal_id, type, title, message, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, user_id, deal_id, type, title, message, is_read, created_at, read_at
	`
	var notif domain.Notification
	err := r.db.QueryRow(ctx, query,
		n.UserID,
		n.DealID,
		n.Type,
		n.Title,
		n.Message,
		n.IsRead,
	).Scan(
		&notif.ID,
		&notif.UserID,
		&notif.DealID,
		&notif.Type,
		&notif.Title,
		&notif.Message,
		&notif.IsRead,
		&notif.CreatedAt,
		&notif.ReadAt,
	)
	if err != nil {
		return nil, err
	}
	return &notif, nil
}

func (r *postgresNotificationRepository) GetNotificationsByUserID(ctx context.Context, userID int, unreadOnly bool) ([]*domain.Notification, error) {
	query := `
		SELECT id, user_id, deal_id, type, title, message, is_read, created_at, read_at
		FROM notifications
		WHERE user_id = $1 AND ($2 = false OR is_read = false)
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID, unreadOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*domain.Notification, 0)
	for rows.Next() {
		var notif domain.Notification
		if err := rows.Scan(
			&notif.ID,
			&notif.UserID,
			&notif.DealID,
			&notif.Type,
			&notif.Title,
			&notif.Message,
			&notif.IsRead,
			&notif.CreatedAt,
			&notif.ReadAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &notif)
	}
	return items, nil
}

func (r *postgresNotificationRepository) MarkAsRead(ctx context.Context, id int, userID int) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = NOW()
		WHERE id = $1 AND user_id = $2
	`
	cmd, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *postgresNotificationRepository) MarkAllAsRead(ctx context.Context, userID int) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = COALESCE(read_at, NOW())
		WHERE user_id = $1 AND is_read = false
	`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}
