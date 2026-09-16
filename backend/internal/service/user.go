package service

import (
	"context"

	"backend/internal/domain"
	"backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	GetUser(ctx context.Context, id int) (*domain.User, error)
	GetAllUsers(ctx context.Context) ([]*domain.User, error)
	GetUsersForEmployee(ctx context.Context, employeeID int) ([]*domain.User, error)
	CanEmployeeAccessUser(ctx context.Context, employeeID, userID int) (bool, error)
	UpdateUser(ctx context.Context, id int, req *domain.UpdateUserRequest) (*domain.User, error)
	UpdateClientPreferences(ctx context.Context, id int, budgetMax *int64, preferences map[string]any) (*domain.ClientPreferences, error)
}

func (s *userService) GetUsersForEmployee(ctx context.Context, employeeID int) ([]*domain.User, error) {
	return s.repo.GetUsersForEmployee(ctx, employeeID)
}

func (s *userService) CanEmployeeAccessUser(ctx context.Context, employeeID, userID int) (bool, error) {
	return s.repo.CanEmployeeAccessUser(ctx, employeeID, userID)
}

func (s *userService) UpdateClientPreferences(
	ctx context.Context,
	id int,
	budgetMax *int64,
	preferences map[string]any,
) (*domain.ClientPreferences, error) {
	return s.repo.UpdateClientPreferences(ctx, id, budgetMax, preferences)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) GetUser(ctx context.Context, id int) (*domain.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *userService) GetAllUsers(ctx context.Context) ([]*domain.User, error) {
	return s.repo.GetAllUsers(ctx)
}

func (s *userService) UpdateUser(ctx context.Context, id int, req *domain.UpdateUserRequest) (*domain.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
