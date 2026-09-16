package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrReminderNotFound = errors.New("reminder not found")

type ReminderRepository interface {
	List(ctx context.Context, assignedTo *int) ([]*domain.StaffReminder, error)
	Create(ctx context.Context, item *domain.StaffReminder) (*domain.StaffReminder, error)
	Complete(ctx context.Context, id int64, actor *domain.User) (*domain.StaffReminder, error)
}

type reminderRepository struct{ db *pgxpool.Pool }

func NewReminderRepository(db *pgxpool.Pool) ReminderRepository { return &reminderRepository{db: db} }
func (r *reminderRepository) List(ctx context.Context, assignedTo *int) ([]*domain.StaffReminder, error) {
	rows, err := r.db.Query(ctx, `SELECT id,assigned_to,created_by,deal_id,title,due_at,completed_at,created_at FROM staff_reminders WHERE ($1::int IS NULL OR assigned_to=$1) ORDER BY completed_at NULLS FIRST,due_at`, assignedTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*domain.StaffReminder, 0)
	for rows.Next() {
		item, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *reminderRepository) Create(ctx context.Context, item *domain.StaffReminder) (*domain.StaffReminder, error) {
	return scanReminder(r.db.QueryRow(ctx, `INSERT INTO staff_reminders (assigned_to,created_by,deal_id,title,due_at) VALUES ($1,$2,$3,$4,$5) RETURNING id,assigned_to,created_by,deal_id,title,due_at,completed_at,created_at`, item.AssignedTo, item.CreatedBy, item.DealID, item.Title, item.DueAt))
}
func (r *reminderRepository) Complete(ctx context.Context, id int64, actor *domain.User) (*domain.StaffReminder, error) {
	item, err := scanReminder(r.db.QueryRow(ctx, `UPDATE staff_reminders SET completed_at=CURRENT_TIMESTAMP WHERE id=$1 AND completed_at IS NULL AND (assigned_to=$2 OR $3='supervisor') RETURNING id,assigned_to,created_by,deal_id,title,due_at,completed_at,created_at`, id, actor.ID, actor.Role))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReminderNotFound
	}
	return item, err
}
func scanReminder(row rowScanner) (*domain.StaffReminder, error) {
	item := new(domain.StaffReminder)
	err := row.Scan(&item.ID, &item.AssignedTo, &item.CreatedBy, &item.DealID, &item.Title, &item.DueAt, &item.CompletedAt, &item.CreatedAt)
	return item, err
}
