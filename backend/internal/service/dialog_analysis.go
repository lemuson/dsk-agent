package service

import (
	"context"
	"sort"

	"backend/internal/domain"
)

type dialogDealReader interface {
	GetDealByID(ctx context.Context, id int) (*domain.Deal, error)
}

type dialogMessageReader interface {
	GetMessagesBySessionID(ctx context.Context, sessionID int) ([]*domain.Message, error)
}

type DialogAnalysisService interface {
	GetDealMessages(ctx context.Context, dealID int, limit int) ([]domain.DealDialogMessage, error)
}

type dialogAnalysisService struct {
	deals    dialogDealReader
	messages dialogMessageReader
}

func NewDialogAnalysisService(deals dialogDealReader, messages dialogMessageReader) DialogAnalysisService {
	return &dialogAnalysisService{deals: deals, messages: messages}
}

func (s *dialogAnalysisService) GetDealMessages(
	ctx context.Context,
	dealID int,
	limit int,
) ([]domain.DealDialogMessage, error) {
	deal, err := s.deals.GetDealByID(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if deal.ChatSessionID == nil {
		return []domain.DealDialogMessage{}, nil
	}

	messages, err := s.messages.GetMessagesBySessionID(ctx, *deal.ChatSessionID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].SendedAt.Equal(messages[j].SendedAt) {
			return messages[i].ID < messages[j].ID
		}
		return messages[i].SendedAt.Before(messages[j].SendedAt)
	})

	result := make([]domain.DealDialogMessage, 0, len(messages))
	for _, message := range messages {
		var direction string
		switch message.SenderType {
		case domain.SenderTypeClient:
			direction = "client_to_manager"
		case domain.SenderTypeManager:
			direction = "manager_to_client"
		default:
			continue
		}
		result = append(result, domain.DealDialogMessage{
			ID:        message.ID,
			Direction: direction,
			Body:      message.Content,
		})
	}
	if len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
}
