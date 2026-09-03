package notification

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Service struct {
	logger *zap.Logger
}

func NewService(logger *zap.Logger) *Service {
	return &Service{logger: logger}
}

type EmailTemplate string

const (
	TemplateBookingConfirmation EmailTemplate = "booking_confirmation"
	TemplateCheckinReminder     EmailTemplate = "checkin_reminder"
	TemplateBoardingPass        EmailTemplate = "boarding_pass"
)

func (s *Service) SendBookingConfirmation(ctx context.Context, pnrID int64) error {
	s.logger.Info("sending booking confirmation email",
		zap.Int64("pnr_id", pnrID),
		zap.Time("sent_at", time.Now()),
	)
	return nil
}

func (s *Service) SendCheckinReminder(ctx context.Context, pnrID int64) error {
	s.logger.Info("sending check-in reminder email",
		zap.Int64("pnr_id", pnrID),
		zap.Time("sent_at", time.Now()),
	)
	return nil
}

func (s *Service) SendBoardingPass(ctx context.Context, pnrID int64) error {
	s.logger.Info("sending boarding pass email",
		zap.Int64("pnr_id", pnrID),
		zap.Time("sent_at", time.Now()),
	)
	return nil
}
