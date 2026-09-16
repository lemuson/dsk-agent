package service

import (
	"context"
	"errors"
	"strings"

	"backend/internal/domain"
	"backend/internal/repository"
)

var (
	ErrInvalidSessionData = errors.New("invalid session data: specify apartment or contact details")
	ErrEmptyMessage       = errors.New("message content cannot be empty")
	ErrEmptyReason        = errors.New("rejection reason cannot be empty")
)

type ChatService interface {
	CreateSession(ctx context.Context, req domain.CreateChatSessionRequest, userID *int) (*domain.ChatSession, error)
	GetSessionByID(ctx context.Context, id int) (*domain.ChatSession, error)
	GetSessions(ctx context.Context, userID *int, employeeID *int, status *domain.ChatSessionStatus) ([]*domain.ChatSession, error)
	TakeSession(ctx context.Context, sessionID int, employeeID int) error
	CloseSession(ctx context.Context, sessionID int) error
	UpdateSessionStatus(ctx context.Context, sessionID int, status domain.ChatSessionStatus) error
	DeleteSessionForUser(ctx context.Context, sessionID int, userID int) error
	RejectSession(ctx context.Context, sessionID int, employeeID int, reason string) (*domain.ChatSessionRejection, error)
	SendMessage(ctx context.Context, sessionID int, userID *int, senderType string, content string) (*domain.Message, error)
	GetMessages(ctx context.Context, sessionID int, currentUserID *int) ([]*domain.Message, error)
}

type chatService struct {
	chatRepo repository.ChatRepository
}

func NewChatService(chatRepo repository.ChatRepository) ChatService {
	return &chatService{chatRepo: chatRepo}
}

func (s *chatService) CreateSession(ctx context.Context, req domain.CreateChatSessionRequest, userID *int) (*domain.ChatSession, error) {

	if userID != nil && req.ApartmentID == nil {
		existing, err := s.chatRepo.GetSessions(ctx, userID, nil, nil)
		if err == nil {
			for _, sess := range existing {
				if sess.ApartmentID == nil && sess.Status != domain.ChatSessionStatusClose {
					if strings.TrimSpace(req.Message) != "" {
						msg := &domain.Message{
							ChatSessionID: sess.ID,
							UserID:        userID,
							SenderType:    "client",
							Content:       req.Message,
							IsRead:        false,
						}
						_, _ = s.chatRepo.CreateMessage(ctx, msg)
					}
					return sess, nil
				}
			}
		}
	}

	session := &domain.ChatSession{
		UserID:      userID,
		ApartmentID: req.ApartmentID,
	}

	if req.GuestName != "" {
		session.GuestName = &req.GuestName
	}
	if req.GuestEmail != "" {
		session.GuestEmail = &req.GuestEmail
	}
	if req.GuestPhone != "" {
		session.GuestPhone = &req.GuestPhone
	}

	created, err := s.chatRepo.CreateSession(ctx, session)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.Message) != "" {
		msg := &domain.Message{
			ChatSessionID: created.ID,
			UserID:        userID,
			SenderType:    "client",
			Content:       req.Message,
			IsRead:        false,
		}
		_, _ = s.chatRepo.CreateMessage(ctx, msg)
	}

	return created, nil
}

func (s *chatService) GetSessionByID(ctx context.Context, id int) (*domain.ChatSession, error) {
	return s.chatRepo.GetSessionByID(ctx, id)
}

func (s *chatService) GetSessions(ctx context.Context, userID *int, employeeID *int, status *domain.ChatSessionStatus) ([]*domain.ChatSession, error) {
	return s.chatRepo.GetSessions(ctx, userID, employeeID, status)
}

func (s *chatService) TakeSession(ctx context.Context, sessionID int, employeeID int) error {
	return s.chatRepo.TakeSession(ctx, sessionID, employeeID)
}

func (s *chatService) CloseSession(ctx context.Context, sessionID int) error {
	return s.chatRepo.CloseSession(ctx, sessionID)
}

func (s *chatService) UpdateSessionStatus(ctx context.Context, sessionID int, status domain.ChatSessionStatus) error {
	return s.chatRepo.UpdateSessionStatus(ctx, sessionID, status)
}

func (s *chatService) DeleteSessionForUser(ctx context.Context, sessionID int, userID int) error {
	return s.chatRepo.DeleteSessionForUser(ctx, sessionID, userID)
}

func (s *chatService) RejectSession(ctx context.Context, sessionID int, employeeID int, reason string) (*domain.ChatSessionRejection, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, ErrEmptyReason
	}
	rejection := &domain.ChatSessionRejection{
		ChatSessionID: sessionID,
		EmployeeID:    employeeID,
		Reason:        strings.TrimSpace(reason),
	}
	return s.chatRepo.CreateRejection(ctx, rejection)
}

func (s *chatService) SendMessage(ctx context.Context, sessionID int, userID *int, senderType string, content string) (*domain.Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyMessage
	}

	_, err := s.chatRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	msg := &domain.Message{
		ChatSessionID: sessionID,
		UserID:        userID,
		SenderType:    senderType,
		Content:       content,
		IsRead:        false,
	}

	return s.chatRepo.CreateMessage(ctx, msg)
}

func (s *chatService) GetMessages(ctx context.Context, sessionID int, currentUserID *int) ([]*domain.Message, error) {

	_, err := s.chatRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	messages, err := s.chatRepo.GetMessagesBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	_ = s.chatRepo.MarkMessagesAsRead(ctx, sessionID, currentUserID)

	return messages, nil
}
