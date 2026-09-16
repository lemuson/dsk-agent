package repository

import (
	"context"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrChatSessionNotFound = errors.New("chat session not found")
	ErrMessageNotFound     = errors.New("message not found")
)

type ChatRepository interface {
	CreateSession(ctx context.Context, session *domain.ChatSession) (*domain.ChatSession, error)
	GetSessionByID(ctx context.Context, id int) (*domain.ChatSession, error)
	GetSessions(ctx context.Context, userID *int, employeeID *int, status *domain.ChatSessionStatus) ([]*domain.ChatSession, error)
	TakeSession(ctx context.Context, sessionID int, employeeID int) error
	CloseSession(ctx context.Context, sessionID int) error
	UpdateSessionStatus(ctx context.Context, sessionID int, status domain.ChatSessionStatus) error
	DeleteSessionForUser(ctx context.Context, sessionID int, userID int) error
	CreateRejection(ctx context.Context, rejection *domain.ChatSessionRejection) (*domain.ChatSessionRejection, error)
	GetRejectionsBySessionID(ctx context.Context, sessionID int) ([]*domain.ChatSessionRejection, error)
	CreateMessage(ctx context.Context, msg *domain.Message) (*domain.Message, error)
	GetMessagesBySessionID(ctx context.Context, sessionID int) ([]*domain.Message, error)
	MarkMessagesAsRead(ctx context.Context, sessionID int, excludingUserID *int) error
}

type chatRepository struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) CreateSession(ctx context.Context, session *domain.ChatSession) (*domain.ChatSession, error) {
	query := `
		INSERT INTO chat_sessions (id_user, id_employee, id_apartment, guest_name, guest_email, guest_phone, status, deleted_by_user, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE, NOW(), NOW())
		RETURNING id, id_user, id_employee, id_apartment, guest_name, guest_email, guest_phone, status, deleted_by_user, created_at, updated_at
	`
	var s domain.ChatSession
	err := r.db.QueryRow(ctx, query,
		session.UserID,
		session.EmployeeID,
		session.ApartmentID,
		session.GuestName,
		session.GuestEmail,
		session.GuestPhone,
		domain.ChatSessionStatusOpen,
	).Scan(
		&s.ID,
		&s.UserID,
		&s.EmployeeID,
		&s.ApartmentID,
		&s.GuestName,
		&s.GuestEmail,
		&s.GuestPhone,
		&s.Status,
		&s.DeletedByUser,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *chatRepository) GetSessionByID(ctx context.Context, id int) (*domain.ChatSession, error) {
	query := `
		SELECT
			cs.id, cs.id_user, cs.id_employee, cs.id_apartment,
			cs.guest_name, cs.guest_email, cs.guest_phone, cs.status, cs.deleted_by_user,
			cs.created_at, cs.updated_at,
			u.name AS user_name,
			e.name AS employee_name
		FROM chat_sessions cs
		LEFT JOIN users u ON cs.id_user = u.id
		LEFT JOIN users e ON cs.id_employee = e.id
		WHERE cs.id = $1
	`
	var s domain.ChatSession
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.UserID,
		&s.EmployeeID,
		&s.ApartmentID,
		&s.GuestName,
		&s.GuestEmail,
		&s.GuestPhone,
		&s.Status,
		&s.DeletedByUser,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.UserName,
		&s.EmployeeName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatSessionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *chatRepository) GetSessions(ctx context.Context, userID *int, employeeID *int, status *domain.ChatSessionStatus) ([]*domain.ChatSession, error) {
	query := `
		SELECT
			cs.id, cs.id_user, cs.id_employee, cs.id_apartment,
			cs.guest_name, cs.guest_email, cs.guest_phone, cs.status, cs.deleted_by_user,
			cs.created_at, cs.updated_at,
			u.name AS user_name,
			e.name AS employee_name
		FROM chat_sessions cs
		LEFT JOIN users u ON cs.id_user = u.id
		LEFT JOIN users e ON cs.id_employee = e.id
		WHERE (CASE
				WHEN $1::int IS NOT NULL THEN cs.id_user = $1 AND cs.deleted_by_user = FALSE
				ELSE NOT (cs.deleted_by_user = TRUE AND cs.id_employee IS NULL)
			   END)
		  AND ($2::int IS NULL OR cs.id_employee = $2)
		  AND ($3::text IS NULL OR cs.status::text = $3)
		ORDER BY cs.updated_at DESC
	`
	var statusStr *string
	if status != nil {
		s := string(*status)
		statusStr = &s
	}

	rows, err := r.db.Query(ctx, query, userID, employeeID, statusStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.ChatSession
	for rows.Next() {
		var s domain.ChatSession
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.EmployeeID,
			&s.ApartmentID,
			&s.GuestName,
			&s.GuestEmail,
			&s.GuestPhone,
			&s.Status,
			&s.DeletedByUser,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.UserName,
			&s.EmployeeName,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}

	if sessions == nil {
		sessions = []*domain.ChatSession{}
	}

	return sessions, nil
}

func (r *chatRepository) TakeSession(ctx context.Context, sessionID int, employeeID int) error {
	query := `
		UPDATE chat_sessions
		SET id_employee = $1, status = $2, updated_at = NOW()
		WHERE id = $3 AND id_employee IS NULL
	`
	cmdTag, err := r.db.Exec(ctx, query, employeeID, domain.ChatSessionStatusInProgress, sessionID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrChatSessionNotFound
	}
	return nil
}

func (r *chatRepository) CloseSession(ctx context.Context, sessionID int) error {
	return r.UpdateSessionStatus(ctx, sessionID, domain.ChatSessionStatusClose)
}

func (r *chatRepository) UpdateSessionStatus(ctx context.Context, sessionID int, status domain.ChatSessionStatus) error {
	query := `
		UPDATE chat_sessions
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, status, sessionID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrChatSessionNotFound
	}
	return nil
}

func (r *chatRepository) DeleteSessionForUser(ctx context.Context, sessionID int, userID int) error {
	query := `
		UPDATE chat_sessions
		SET deleted_by_user = TRUE, updated_at = NOW()
		WHERE id = $1 AND id_user = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, sessionID, userID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrChatSessionNotFound
	}
	return nil
}

func (r *chatRepository) CreateRejection(ctx context.Context, rejection *domain.ChatSessionRejection) (*domain.ChatSessionRejection, error) {
	query := `
		INSERT INTO chat_session_rejections (id_chat_sessions, id_employee, reason, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, id_chat_sessions, id_employee, reason, created_at
	`
	var rej domain.ChatSessionRejection
	err := r.db.QueryRow(ctx, query, rejection.ChatSessionID, rejection.EmployeeID, rejection.Reason).Scan(
		&rej.ID,
		&rej.ChatSessionID,
		&rej.EmployeeID,
		&rej.Reason,
		&rej.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	_ = r.CloseSession(ctx, rejection.ChatSessionID)

	return &rej, nil
}

func (r *chatRepository) GetRejectionsBySessionID(ctx context.Context, sessionID int) ([]*domain.ChatSessionRejection, error) {
	query := `
		SELECT r.id, r.id_chat_sessions, r.id_employee, r.reason, r.created_at, u.name
		FROM chat_session_rejections r
		LEFT JOIN users u ON r.id_employee = u.id
		WHERE r.id_chat_sessions = $1
		ORDER BY r.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ChatSessionRejection
	for rows.Next() {
		var item domain.ChatSessionRejection
		if err := rows.Scan(&item.ID, &item.ChatSessionID, &item.EmployeeID, &item.Reason, &item.CreatedAt, &item.EmployeeName); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}
	if list == nil {
		list = []*domain.ChatSessionRejection{}
	}
	return list, nil
}

func (r *chatRepository) CreateMessage(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	query := `
		INSERT INTO messages (id_chat_session, id_user, sender_type, content, is_read, sended_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, id_chat_session, id_user, sender_type, content, is_read, sended_at
	`
	var m domain.Message
	err := r.db.QueryRow(ctx, query, msg.ChatSessionID, msg.UserID, msg.SenderType, msg.Content, msg.IsRead).Scan(
		&m.ID,
		&m.ChatSessionID,
		&m.UserID,
		&m.SenderType,
		&m.Content,
		&m.IsRead,
		&m.SendedAt,
	)
	if err != nil {
		return nil, err
	}

	_, _ = r.db.Exec(ctx, `UPDATE chat_sessions SET updated_at = NOW() WHERE id = $1`, msg.ChatSessionID)

	return &m, nil
}

func (r *chatRepository) GetMessagesBySessionID(ctx context.Context, sessionID int) ([]*domain.Message, error) {
	query := `
		SELECT m.id, m.id_chat_session, m.id_user, m.sender_type, m.content, m.is_read, m.sended_at, u.name
		FROM messages m
		LEFT JOIN users u ON m.id_user = u.id
		WHERE m.id_chat_session = $1
		ORDER BY m.sended_at ASC
	`
	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var m domain.Message
		err := rows.Scan(
			&m.ID,
			&m.ChatSessionID,
			&m.UserID,
			&m.SenderType,
			&m.Content,
			&m.IsRead,
			&m.SendedAt,
			&m.SenderName,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}
	if messages == nil {
		messages = []*domain.Message{}
	}
	return messages, nil
}

func (r *chatRepository) MarkMessagesAsRead(ctx context.Context, sessionID int, excludingUserID *int) error {
	query := `
		UPDATE messages
		SET is_read = TRUE
		WHERE id_chat_session = $1
		  AND ($2::int IS NULL OR id_user IS NULL OR id_user != $2)
		  AND is_read = FALSE
	`
	_, err := r.db.Exec(ctx, query, sessionID, excludingUserID)
	return err
}
