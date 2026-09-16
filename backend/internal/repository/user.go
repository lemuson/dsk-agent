package repository

import (
	"context"
	"encoding/json"
	"errors"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	CreateUser(ctx context.Context, name, email, passwordHash string, role domain.Role) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id int) (*domain.User, error)
	GetAllUsers(ctx context.Context) ([]*domain.User, error)
	GetUsersForEmployee(ctx context.Context, employeeID int) ([]*domain.User, error)
	CanEmployeeAccessUser(ctx context.Context, employeeID, userID int) (bool, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	UpdateClientPreferences(ctx context.Context, id int, budgetMax *int64, preferences map[string]any) (*domain.ClientPreferences, error)
}

func (r *userRepository) GetUsersForEmployee(ctx context.Context, employeeID int) ([]*domain.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, email, role, budget_max, preferences
		FROM users u
		WHERE u.role = 'user' AND (
			EXISTS (SELECT 1 FROM deals d WHERE d.id_user = u.id AND d.id_employee = $1)
			OR EXISTS (SELECT 1 FROM chat_sessions cs WHERE cs.id_user = u.id AND cs.id_employee = $1)
		)
		ORDER BY name, id
	`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]*domain.User, 0)
	for rows.Next() {
		var user domain.User
		var rawPreferences []byte
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.BudgetMax, &rawPreferences); err != nil {
			return nil, err
		}
		user.Preferences, err = domain.DecodePreferences(rawPreferences)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

func (r *userRepository) CanEmployeeAccessUser(ctx context.Context, employeeID, userID int) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM deals WHERE id_user = $2 AND id_employee = $1
			UNION ALL
			SELECT 1 FROM chat_sessions WHERE id_user = $2 AND id_employee = $1
		)
	`, employeeID, userID).Scan(&allowed)
	return allowed, err
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, name, email, passwordHash string, role domain.Role) (*domain.User, error) {
	query := `INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING id, name, email, role`
	var user domain.User
	err := r.db.QueryRow(ctx, query, name, email, passwordHash, role).Scan(&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, name, email, password_hash, role FROM users WHERE email = $1`
	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT id, name, email, password_hash, role, budget_max, preferences FROM users WHERE id = $1`
	var user domain.User
	var rawPreferences []byte
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role, &user.BudgetMax, &rawPreferences)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	user.Preferences, err = domain.DecodePreferences(rawPreferences)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetAllUsers(ctx context.Context) ([]*domain.User, error) {
	query := `SELECT id, name, email, role FROM users`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

func (r *userRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `UPDATE users SET name = $1, email = $2, password_hash = $3 WHERE id = $4`
	_, err := r.db.Exec(ctx, query, user.Name, user.Email, user.PasswordHash, user.ID)
	return err
}

func (r *userRepository) UpdateClientPreferences(
	ctx context.Context,
	id int,
	budgetMax *int64,
	preferences map[string]any,
) (*domain.ClientPreferences, error) {
	patch, err := json.Marshal(preferences)
	if err != nil {
		return nil, err
	}
	query := `
		UPDATE users
		SET budget_max = COALESCE($2, budget_max),
		    preferences = preferences || $3::jsonb
		WHERE id = $1
		RETURNING id, budget_max, preferences
	`
	var result domain.ClientPreferences
	var rawPreferences []byte
	err = r.db.QueryRow(ctx, query, id, budgetMax, patch).Scan(
		&result.ClientID,
		&result.BudgetMax,
		&rawPreferences,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	result.Preferences, err = domain.DecodePreferences(rawPreferences)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
