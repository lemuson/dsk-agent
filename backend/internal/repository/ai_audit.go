package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AIAuditRepository interface {
	Record(ctx context.Context, actorID int, dealID, sessionID *int, action, requestText, responseText, agent, intent, status string) error
}

type aiAuditRepository struct{ db *pgxpool.Pool }

func NewAIAuditRepository(db *pgxpool.Pool) AIAuditRepository { return &aiAuditRepository{db: db} }

func (r *aiAuditRepository) Record(ctx context.Context, actorID int, dealID, sessionID *int, action, requestText, responseText, agent, intent, status string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO ai_audit_log (actor_id,deal_id,chat_session_id,action,request_text,response_text,agent,intent,status) VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),$9)`, actorID, dealID, sessionID, action, requestText, responseText, agent, intent, status)
	return err
}
