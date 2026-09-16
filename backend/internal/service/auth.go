package service

import (
	"context"
	"errors"
	"time"

	"backend/internal/domain"
	"backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
)

type AuthService interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, error)
	RegisterStaff(ctx context.Context, name, email, password string, role domain.Role) (*domain.User, error)
	Login(ctx context.Context, email, password string) (string, *domain.User, error)
	ValidateToken(tokenString string) (*domain.User, error)
	GenerateToken(user *domain.User) (string, error)
}

type authService struct {
	repo      repository.UserRepository
	jwtSecret []byte
}

func NewAuthService(repo repository.UserRepository, secret string) AuthService {
	return &authService{
		repo:      repo,
		jwtSecret: []byte(secret),
	}
}

func (s *authService) Register(ctx context.Context, name, email, password string) (*domain.User, error) {

	_, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserExists
	}
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.repo.CreateUser(ctx, name, email, string(hash), domain.RoleUser)
}

func (s *authService) RegisterStaff(ctx context.Context, name, email, password string, role domain.Role) (*domain.User, error) {

	_, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserExists
	}
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.repo.CreateUser(ctx, name, email, string(hash), role)
}

func (s *authService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	tokenString, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, err
	}

	return tokenString, user, nil
}

func (s *authService) GenerateToken(user *domain.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString(s.jwtSecret)
}

func (s *authService) ValidateToken(tokenString string) (*domain.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	id := int(claims["id"].(float64))
	name := claims["name"].(string)
	email := claims["email"].(string)
	role := claims["role"].(string)

	return &domain.User{
		ID:    id,
		Name:  name,
		Email: email,
		Role:  domain.Role(role),
	}, nil
}
