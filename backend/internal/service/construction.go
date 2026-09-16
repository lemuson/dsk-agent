package service

import (
	"context"

	"backend/internal/domain"
	"backend/internal/repository"
)

type ConstructionService interface {

	CreateResidentialComplex(ctx context.Context, complex *domain.ResidentialComplex) error
	GetResidentialComplexByID(ctx context.Context, id int) (*domain.ResidentialComplex, error)
	GetAllResidentialComplexes(ctx context.Context) ([]*domain.ResidentialComplex, error)
	UpdateResidentialComplex(ctx context.Context, complex *domain.ResidentialComplex) error
	DeleteResidentialComplex(ctx context.Context, id int) error

	CreateBuilding(ctx context.Context, building *domain.Building) error
	GetBuildingByID(ctx context.Context, id int) (*domain.Building, error)
	GetBuildingsByComplexID(ctx context.Context, complexID int) ([]*domain.Building, error)
	UpdateBuilding(ctx context.Context, building *domain.Building) error
	DeleteBuilding(ctx context.Context, id int) error

	CreateApartment(ctx context.Context, apartment *domain.Apartment) error
	GetApartmentByID(ctx context.Context, id int) (*domain.Apartment, error)
	GetApartmentsByBuildingID(ctx context.Context, buildingID int) ([]*domain.Apartment, error)
	UpdateApartment(ctx context.Context, apartment *domain.Apartment) error
	DeleteApartment(ctx context.Context, id int) error

	CreateProgress(ctx context.Context, progress *domain.ConstructionProgress) error
	GetProgressByID(ctx context.Context, id int) (*domain.ConstructionProgress, error)
	GetProgressByBuildingID(ctx context.Context, buildingID int) ([]*domain.ConstructionProgress, error)
	UpdateProgress(ctx context.Context, progress *domain.ConstructionProgress) error
	DeleteProgress(ctx context.Context, id int) error
}

type constructionService struct {
	repo         repository.ConstructionRepository
	notification NotificationService
}

func NewConstructionService(repo repository.ConstructionRepository, notification NotificationService) ConstructionService {
	return &constructionService{
		repo:         repo,
		notification: notification,
	}
}

func (s *constructionService) CreateResidentialComplex(ctx context.Context, complex *domain.ResidentialComplex) error {
	return s.repo.CreateResidentialComplex(ctx, complex)
}

func (s *constructionService) GetResidentialComplexByID(ctx context.Context, id int) (*domain.ResidentialComplex, error) {
	return s.repo.GetResidentialComplexByID(ctx, id)
}

func (s *constructionService) GetAllResidentialComplexes(ctx context.Context) ([]*domain.ResidentialComplex, error) {
	return s.repo.GetAllResidentialComplexes(ctx)
}

func (s *constructionService) UpdateResidentialComplex(ctx context.Context, complex *domain.ResidentialComplex) error {
	return s.repo.UpdateResidentialComplex(ctx, complex)
}

func (s *constructionService) DeleteResidentialComplex(ctx context.Context, id int) error {
	return s.repo.DeleteResidentialComplex(ctx, id)
}

func (s *constructionService) CreateBuilding(ctx context.Context, building *domain.Building) error {
	return s.repo.CreateBuilding(ctx, building)
}

func (s *constructionService) GetBuildingByID(ctx context.Context, id int) (*domain.Building, error) {
	return s.repo.GetBuildingByID(ctx, id)
}

func (s *constructionService) GetBuildingsByComplexID(ctx context.Context, complexID int) ([]*domain.Building, error) {
	return s.repo.GetBuildingsByComplexID(ctx, complexID)
}

func (s *constructionService) UpdateBuilding(ctx context.Context, building *domain.Building) error {
	return s.repo.UpdateBuilding(ctx, building)
}

func (s *constructionService) DeleteBuilding(ctx context.Context, id int) error {
	return s.repo.DeleteBuilding(ctx, id)
}

func (s *constructionService) CreateApartment(ctx context.Context, apartment *domain.Apartment) error {
	return s.repo.CreateApartment(ctx, apartment)
}

func (s *constructionService) GetApartmentByID(ctx context.Context, id int) (*domain.Apartment, error) {
	return s.repo.GetApartmentByID(ctx, id)
}

func (s *constructionService) GetApartmentsByBuildingID(ctx context.Context, buildingID int) ([]*domain.Apartment, error) {
	return s.repo.GetApartmentsByBuildingID(ctx, buildingID)
}

func (s *constructionService) UpdateApartment(ctx context.Context, apartment *domain.Apartment) error {
	return s.repo.UpdateApartment(ctx, apartment)
}

func (s *constructionService) DeleteApartment(ctx context.Context, id int) error {
	return s.repo.DeleteApartment(ctx, id)
}

func (s *constructionService) CreateProgress(ctx context.Context, progress *domain.ConstructionProgress) error {
	if err := s.repo.CreateProgress(ctx, progress); err != nil {
		return err
	}
	if s.notification != nil {
		_ = s.notification.NotifyConstructionUpdate(ctx, progress)
	}
	return nil
}

func (s *constructionService) GetProgressByID(ctx context.Context, id int) (*domain.ConstructionProgress, error) {
	return s.repo.GetProgressByID(ctx, id)
}

func (s *constructionService) GetProgressByBuildingID(ctx context.Context, buildingID int) ([]*domain.ConstructionProgress, error) {
	return s.repo.GetProgressByBuildingID(ctx, buildingID)
}

func (s *constructionService) UpdateProgress(ctx context.Context, progress *domain.ConstructionProgress) error {
	if err := s.repo.UpdateProgress(ctx, progress); err != nil {
		return err
	}
	if s.notification != nil {
		_ = s.notification.NotifyConstructionUpdate(ctx, progress)
	}
	return nil
}

func (s *constructionService) DeleteProgress(ctx context.Context, id int) error {
	return s.repo.DeleteProgress(ctx, id)
}
