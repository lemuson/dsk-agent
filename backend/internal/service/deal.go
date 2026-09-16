package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"backend/internal/domain"
	"backend/internal/repository"
)

var (
	ErrInvalidDealData          = errors.New("invalid deal data")
	ErrDealForbidden            = errors.New("deal operation forbidden")
	ErrApartmentUnavailable     = errors.New("apartment is not available")
	ErrDiscountApprovalRequired = errors.New("discount requires approval")
)

type DealService interface {
	CreateDeal(ctx context.Context, req domain.CreateDealRequest, actor *domain.User) (*domain.Deal, error)
	GetDealByID(ctx context.Context, id int) (*domain.Deal, error)
	GetDeals(ctx context.Context, userID *int, employeeID *int, status *domain.DealStatus) ([]*domain.Deal, error)
	UpdateDealStatus(ctx context.Context, id int, req domain.UpdateDealStatusRequest, actor *domain.User) (*domain.Deal, error)
	GetDealsByBuildingID(ctx context.Context, buildingID int) ([]*domain.Deal, error)
}

func (s *dealService) GetDealsByBuildingID(ctx context.Context, buildingID int) ([]*domain.Deal, error) {
	return s.dealRepo.GetDealsByBuildingID(ctx, buildingID)
}

type dealService struct {
	dealRepo         repository.DealRepository
	policyReader     discountPolicyReader
	notificationRepo repository.NotificationRepository
	userRepo         repository.UserRepository
	chatRepo         repository.ChatRepository
	emailSender      EmailSender
	managerMax       float64
	supervisorMax    float64
}

type discountPolicyReader interface {
	GetActiveMax(ctx context.Context, apartmentID int, role domain.Role) (string, error)
}

func NewDealService(dealRepo repository.DealRepository, managerMax, supervisorMax float64, readers ...discountPolicyReader) *dealService {
	var reader discountPolicyReader
	if len(readers) > 0 {
		reader = readers[0]
	}
	return &dealService{dealRepo: dealRepo, policyReader: reader, managerMax: managerMax, supervisorMax: supervisorMax}
}

func (s *dealService) SetNotificationDependencies(notifRepo repository.NotificationRepository, userRepo repository.UserRepository, chatRepo repository.ChatRepository, emailSender EmailSender) {
	s.notificationRepo = notifRepo
	s.userRepo = userRepo
	s.chatRepo = chatRepo
	s.emailSender = emailSender
}

func roundPrice(val float64) float64 {
	return math.Round(val*100) / 100
}

func (s *dealService) CreateDeal(ctx context.Context, req domain.CreateDealRequest, actor *domain.User) (*domain.Deal, error) {
	if actor == nil || (actor.Role != domain.RoleManager && actor.Role != domain.RoleSupervisor) {
		return nil, ErrDealForbidden
	}
	maxDiscount, err := s.maxDiscount(ctx, req.ApartmentID, actor.Role)
	if err != nil {
		return nil, err
	}
	if req.UserID <= 0 || req.ApartmentID <= 0 || req.PercentDiscount < 0 || req.PercentDiscount > maxDiscount {
		if req.PercentDiscount > maxDiscount {
			return nil, ErrDiscountApprovalRequired
		}
		return nil, ErrInvalidDealData
	}
	allowed, err := s.dealRepo.CanCreateDealForUser(ctx, actor, req.UserID, req.ChatSessionID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrDealForbidden
	}
	basePrice, status, err := s.dealRepo.GetApartmentSaleData(ctx, req.ApartmentID)
	if err != nil {
		return nil, err
	}
	if basePrice <= 0 || status != domain.ApartmentStatusFree {
		return nil, ErrApartmentUnavailable
	}

	totalPrice := roundPrice(basePrice * (1.0 - req.PercentDiscount/100.0))

	deal := &domain.Deal{
		UserID:          req.UserID,
		EmployeeID:      actor.ID,
		ApartmentID:     req.ApartmentID,
		ChatSessionID:   req.ChatSessionID,
		BasePrice:       basePrice,
		PercentDiscount: req.PercentDiscount,
		TotalPrice:      totalPrice,
	}

	createdDeal, err := s.dealRepo.CreateDeal(ctx, deal)
	if err != nil {
		return nil, err
	}

	if s.notificationRepo != nil {
		notif := &domain.Notification{
			UserID:  createdDeal.UserID,
			DealID:  &createdDeal.ID,
			Type:    domain.NotificationTypeDealUpdate,
			Title:   fmt.Sprintf("Сформирована сделка #%d", createdDeal.ID),
			Message: fmt.Sprintf("Менеджер оформил сделку #%d по выбранной квартире. Статус: «На согласовании». Квартира забронирована.", createdDeal.ID),
			IsRead:  false,
		}
		_, _ = s.notificationRepo.CreateNotification(ctx, notif)
	}

	if s.emailSender != nil && s.userRepo != nil {
		client, userErr := s.userRepo.GetUserByID(ctx, createdDeal.UserID)
		if userErr == nil && client != nil && strings.TrimSpace(client.Email) != "" {
			subject := fmt.Sprintf("Оформление сделки #%d — АО СЗ «ДСК»", createdDeal.ID)
			body := fmt.Sprintf("Здравствуйте, %s!\n\nМенеджер оформил сделку #%d. Статус: «На согласовании». Квартира успешно забронирована.\nИтоговая сумма: %.2f руб.\n\nВы можете ознакомиться с условиями и согласовать сделку в вашем личном кабинете.\n\nС уважением,\nОтдел продаж АО СЗ «ДСК»", client.Name, createdDeal.ID, createdDeal.TotalPrice)
			_ = s.emailSender.SendEmail(client.Email, subject, body)
		}
	}

	if s.chatRepo != nil && createdDeal.ChatSessionID != nil {
		sysMsg := &domain.Message{
			ChatSessionID: *createdDeal.ChatSessionID,
			SenderType:    "system",
			Content:       fmt.Sprintf("Сформирована сделка #%d на сумму %.2f руб. Статус заявки и сделки переведен в «На согласовании». Квартира забронирована.", createdDeal.ID, createdDeal.TotalPrice),
			IsRead:        false,
		}
		_, _ = s.chatRepo.CreateMessage(ctx, sysMsg)
	}

	return createdDeal, nil
}

func (s *dealService) GetDealByID(ctx context.Context, id int) (*domain.Deal, error) {
	return s.dealRepo.GetDealByID(ctx, id)
}

func (s *dealService) GetDeals(ctx context.Context, userID *int, employeeID *int, status *domain.DealStatus) ([]*domain.Deal, error) {
	return s.dealRepo.GetDeals(ctx, userID, employeeID, status)
}

func (s *dealService) UpdateDealStatus(ctx context.Context, id int, req domain.UpdateDealStatusRequest, actor *domain.User) (*domain.Deal, error) {
	existing, err := s.dealRepo.GetDealByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, ErrDealForbidden
	}

	if actor.Role == domain.RoleUser {
		if existing.UserID != actor.ID {
			return nil, ErrDealForbidden
		}
		if req.Status != domain.DealStatusContract && req.Status != domain.DealStatusCancelled {
			return nil, ErrInvalidDealData
		}
	} else if actor.Role == domain.RoleManager {
		if existing.EmployeeID != actor.ID {
			return nil, ErrDealForbidden
		}
	} else if actor.Role != domain.RoleSupervisor {
		return nil, ErrDealForbidden
	}

	if req.Status != domain.DealStatusPending && req.Status != domain.DealStatusContract && req.Status != domain.DealStatusCompleted && req.Status != domain.DealStatusCancelled {
		return nil, ErrInvalidDealData
	}

	var discount *float64
	var totalPrice *float64

	if req.PercentDiscount != nil {
		d := *req.PercentDiscount
		maxDiscount, policyErr := s.maxDiscount(ctx, existing.ApartmentID, actor.Role)
		if policyErr != nil {
			return nil, policyErr
		}
		if d < 0 || d > maxDiscount {
			if d > maxDiscount {
				return nil, ErrDiscountApprovalRequired
			}
			return nil, ErrInvalidDealData
		}
		discount = &d
		tp := roundPrice(existing.BasePrice * (1.0 - d/100.0))
		totalPrice = &tp
	}

	updatedDeal, err := s.dealRepo.UpdateDealStatus(ctx, id, req.Status, discount, totalPrice)
	if err != nil {
		return nil, err
	}

	if updatedDeal.ChatSessionID != nil && s.chatRepo != nil {
		var content string
		switch req.Status {
		case domain.DealStatusContract:
			content = fmt.Sprintf("Клиент подтвердил сделку #%d. Статус заявки изменен на «Согласовано» (Договор).", updatedDeal.ID)
		case domain.DealStatusCompleted:
			content = fmt.Sprintf("Сделка #%d успешно закрыта и завершена. Квартира переведена в статус «Продано».", updatedDeal.ID)
		case domain.DealStatusPending:
			content = fmt.Sprintf("Предложение по сделке #%d возвращено в работу для корректировки условий.", updatedDeal.ID)
		case domain.DealStatusCancelled:
			content = fmt.Sprintf("Сделка #%d отменена.", updatedDeal.ID)
		}
		if content != "" {
			sysMsg := &domain.Message{
				ChatSessionID: *updatedDeal.ChatSessionID,
				SenderType:    "system",
				Content:       content,
				IsRead:        false,
			}
			_, _ = s.chatRepo.CreateMessage(ctx, sysMsg)
		}
	}

	return updatedDeal, nil
}

func (s *dealService) maxDiscount(ctx context.Context, apartmentID int, role domain.Role) (float64, error) {
	if s.policyReader != nil {
		value, err := s.policyReader.GetActiveMax(ctx, apartmentID, role)
		if err == nil {
			parsed, parseErr := strconv.ParseFloat(value, 64)
			if parseErr != nil {
				return 0, parseErr
			}
			return parsed, nil
		}
		if !errors.Is(err, repository.ErrDiscountPolicyNotFound) {
			return 0, err
		}
	}
	if role == domain.RoleSupervisor {
		return s.supervisorMax, nil
	}
	return s.managerMax, nil
}
