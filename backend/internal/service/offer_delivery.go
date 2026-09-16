package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"

	"backend/internal/config"
	"backend/internal/domain"
	"backend/internal/repository"
)

var ErrOfferEmailNotConfigured = errors.New("SMTP delivery is not configured")

type OfferDeliveryService interface {
	Store(ctx context.Context, offerID int, pdf []byte) error
	Send(ctx context.Context, offer *domain.Offer, pdf []byte) (*domain.OfferDelivery, error)
}

func (s *offerDeliveryService) Store(ctx context.Context, offerID int, pdf []byte) error {
	checksum := fmt.Sprintf("%x", sha256.Sum256(pdf))
	return s.repository.SaveDocument(ctx, offerID, pdf, checksum)
}

type offerAttachmentSender interface {
	Send(to, subject, body, filename string, attachment []byte) error
}

type offerDeliveryService struct {
	repository repository.OfferDeliveryRepository
	deals      offerDealReader
	users      offerUserReader
	sender     offerAttachmentSender
}

func NewOfferDeliveryService(repository repository.OfferDeliveryRepository, deals offerDealReader, users offerUserReader, sender offerAttachmentSender) OfferDeliveryService {
	return &offerDeliveryService{repository: repository, deals: deals, users: users, sender: sender}
}

func (s *offerDeliveryService) Send(ctx context.Context, offer *domain.Offer, pdf []byte) (*domain.OfferDelivery, error) {
	if offer == nil || offer.Status != domain.OfferStatusApproved {
		return nil, ErrInvalidOfferState
	}
	deal, err := s.deals.GetDealByID(ctx, offer.DealID)
	if err != nil {
		return nil, err
	}
	client, err := s.users.GetUserByID(ctx, deal.UserID)
	if err != nil {
		return nil, err
	}
	if err := s.Store(ctx, offer.ID, pdf); err != nil {
		return nil, err
	}
	delivery, err := s.repository.CreateDelivery(ctx, offer.ID, client.Email)
	if err != nil {
		return nil, err
	}
	body := fmt.Sprintf("Здравствуйте, %s!\n\nНаправляем утверждённое коммерческое предложение АО СЗ «ДСК».\n\n%s", client.Name, offer.GeneratedText)
	if err := s.sender.Send(client.Email, "Коммерческое предложение ДСК", body, fmt.Sprintf("dsk-offer-%d.pdf", offer.ID), pdf); err != nil {
		failed, finishErr := s.repository.FinishDelivery(ctx, delivery.ID, false, err.Error())
		if finishErr != nil {
			return nil, finishErr
		}
		return failed, err
	}
	return s.repository.FinishDelivery(ctx, delivery.ID, true, "")
}

type smtpOfferAttachmentSender struct{ host, port, username, password, from string }

func NewSMTPOfferAttachmentSender(cfg *config.Config) offerAttachmentSender {
	return &smtpOfferAttachmentSender{host: cfg.SMTPHost, port: cfg.SMTPPort, username: cfg.SMTPUsername, password: cfg.SMTPPassword, from: cfg.SMTPFrom}
}

func (s *smtpOfferAttachmentSender) Send(to, subject, body, filename string, attachment []byte) error {
	if strings.TrimSpace(s.host) == "" || strings.TrimSpace(s.port) == "" {
		return ErrOfferEmailNotConfigured
	}
	from := s.from
	if from == "" {
		from = "no-reply@dsk-agent.ru"
	}
	var message bytes.Buffer
	writer := multipart.NewWriter(&message)
	fmt.Fprintf(&message, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%q\r\n\r\n", from, to, mime.QEncoding.Encode("UTF-8", subject), writer.Boundary())
	bodyHeader := textproto.MIMEHeader{"Content-Type": {"text/plain; charset=UTF-8"}, "Content-Transfer-Encoding": {"8bit"}}
	bodyPart, _ := writer.CreatePart(bodyHeader)
	_, _ = bodyPart.Write([]byte(body))
	attachmentHeader := textproto.MIMEHeader{"Content-Type": {"application/pdf"}, "Content-Disposition": {fmt.Sprintf(`attachment; filename="%s"`, filename)}, "Content-Transfer-Encoding": {"base64"}}
	attachmentPart, _ := writer.CreatePart(attachmentHeader)
	encoded := base64.StdEncoding.EncodeToString(attachment)
	for len(encoded) > 76 {
		_, _ = fmt.Fprintf(attachmentPart, "%s\r\n", encoded[:76])
		encoded = encoded[76:]
	}
	_, _ = fmt.Fprint(attachmentPart, encoded)
	_ = writer.Close()
	var auth smtp.Auth
	if s.username != "" && s.password != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	return smtp.SendMail(s.host+":"+s.port, auth, from, []string{to}, message.Bytes())
}
