package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"backend/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrRPCResponseTimeout  = errors.New("agent service response timed out")
	ErrRPCFailed           = errors.New("agent service returned error")
	ErrRabbitMQUnavailable = errors.New("rabbitmq broker is currently unavailable")
)

type AgentRPCError struct {
	Code    string
	Message string
}

func (e *AgentRPCError) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

func (e *AgentRPCError) Unwrap() error { return ErrRPCFailed }

var idCounter uint64

func generateSimpleID(prefix string) string {
	counter := atomic.AddUint64(&idCounter, 1)
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)
	return fmt.Sprintf("%s_%d_%d_%s", prefix, time.Now().UnixMilli(), counter, hex.EncodeToString(randomBytes))
}

type AIAgentService interface {
	SendChatRequest(ctx context.Context, payload domain.AgentChatPayload) (*domain.AgentChatResponseData, error)
	AnalyzeDialog(ctx context.Context, dealID, userID int) (*domain.DialogAnalyzeResponseData, error)
	AssistDialogReply(ctx context.Context, dealID, userID int, selectedText *string) (*domain.DialogReplyAssistResponseData, error)
	NewRabbitMQChannel() (*amqp.Channel, error)
	Close() error
}

type aiAgentService struct {
	rabbitURL string
	mu        sync.Mutex
	conn      *amqp.Connection
	ch        *amqp.Channel
}

func NewAIAgentService(rabbitURL string) AIAgentService {
	return &aiAgentService{
		rabbitURL: rabbitURL,
	}
}

func (s *aiAgentService) ensureConnection() (*amqp.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil && !s.conn.IsClosed() && s.ch != nil && !s.ch.IsClosed() {
		return s.ch, nil
	}

	conn, err := amqp.Dial(s.rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRabbitMQUnavailable, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open channel: %v", err)
	}

	err = ch.ExchangeDeclare(
		"app.topic",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare exchange app.topic: %v", err)
	}

	s.conn = conn
	s.ch = ch
	return s.ch, nil
}

func (s *aiAgentService) SendChatRequest(ctx context.Context, payload domain.AgentChatPayload) (*domain.AgentChatResponseData, error) {
	var response domain.AgentChatResponseData
	if err := s.sendRPC(ctx, "agent.chat.request", "chat", payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *aiAgentService) AnalyzeDialog(ctx context.Context, dealID, userID int) (*domain.DialogAnalyzeResponseData, error) {
	var response domain.DialogAnalyzeResponseData
	payload := map[string]any{"deal_id": dealID, "user_id": userID}
	if err := s.sendRPC(ctx, "agent.dialog.analyze", "dialog.analyze", payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *aiAgentService) AssistDialogReply(ctx context.Context, dealID, userID int, selectedText *string) (*domain.DialogReplyAssistResponseData, error) {
	var response domain.DialogReplyAssistResponseData
	payload := map[string]any{"deal_id": dealID, "user_id": userID, "selected_text": selectedText}
	if err := s.sendRPC(ctx, "agent.dialog.reply_assist", "dialog.reply_assist", payload, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

type agentRPCEnvelope struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
	Payload   any    `json:"payload"`
}

type agentRPCResponse struct {
	RequestID string                 `json:"request_id"`
	Success   bool                   `json:"success"`
	Data      json.RawMessage        `json:"data"`
	Error     *domain.AgentChatError `json:"error"`
}

func (s *aiAgentService) sendRPC(ctx context.Context, routingKey, action string, payload, output any) error {
	ch, err := s.ensureConnection()
	if err != nil {
		return err
	}

	requestID := generateSimpleID("req")
	correlationID := generateSimpleID("corr")

	reqEnvelope := agentRPCEnvelope{
		RequestID: requestID,
		Action:    action,
		Payload:   payload,
	}

	body, err := json.Marshal(reqEnvelope)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	q, err := ch.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare callback queue: %v", err)
	}
	defer func() {
		_, _ = ch.QueueDelete(q.Name, false, false, false)
	}()

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %v", err)
	}

	err = ch.PublishWithContext(
		ctx,
		"app.topic",
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       q.Name,
			Body:          body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	timeout := 75 * time.Second
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return ErrRPCResponseTimeout
		case d, ok := <-msgs:
			if !ok {
				return errors.New("callback queue channel closed")
			}

			if d.CorrelationId != correlationID {
				continue
			}

			var respEnvelope agentRPCResponse
			if err := json.Unmarshal(d.Body, &respEnvelope); err != nil {
				return fmt.Errorf("failed to parse agent response: %v", err)
			}

			if respEnvelope.RequestID != requestID {
				return fmt.Errorf("request_id mismatch: expected %s, got %s", requestID, respEnvelope.RequestID)
			}

			if !respEnvelope.Success {
				if respEnvelope.Error == nil {
					return ErrRPCFailed
				}
				return &AgentRPCError{
					Code:    respEnvelope.Error.Code,
					Message: respEnvelope.Error.Message,
				}
			}

			if len(respEnvelope.Data) == 0 || string(respEnvelope.Data) == "null" {
				return errors.New("agent response data is empty")
			}
			if err := json.Unmarshal(respEnvelope.Data, output); err != nil {
				return fmt.Errorf("failed to parse agent response data: %v", err)
			}
			return nil
		}
	}
}

func (s *aiAgentService) NewRabbitMQChannel() (*amqp.Channel, error) {
	if _, err := s.ensureConnection(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil || s.conn.IsClosed() {
		return nil, ErrRabbitMQUnavailable
	}
	return s.conn.Channel()
}

func (s *aiAgentService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ch != nil && !s.ch.IsClosed() {
		_ = s.ch.Close()
	}
	if s.conn != nil && !s.conn.IsClosed() {
		return s.conn.Close()
	}
	return nil
}
