package handler

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/api/helper"
	"github.com/you/airline-ticketing/internal/domain/notification"
)

type NotificationHandler struct {
	svc *notification.Service
	log *zap.Logger
}

func NewNotificationHandler(svc *notification.Service, log *zap.Logger) *NotificationHandler {
	return &NotificationHandler{svc: svc, log: log}
}

type sendNotificationRequest struct {
	PNRID int64  `json:"pnr_id" validate:"required,gt=0"`
	Type  string `json:"type" validate:"required"`
}

// Send accepts a manual notification trigger. It is primarily a
// development aid since real delivery (email/SMS) is not wired up.
func (h *NotificationHandler) Send(w http.ResponseWriter, r *http.Request) {
	var in sendNotificationRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	var err error
	switch in.Type {
	case string(notification.TemplateBookingConfirmation):
		err = h.svc.SendBookingConfirmation(r.Context(), in.PNRID)
	case string(notification.TemplateCheckinReminder):
		err = h.svc.SendCheckinReminder(r.Context(), in.PNRID)
	case string(notification.TemplateBoardingPass):
		err = h.svc.SendBoardingPass(r.Context(), in.PNRID)
	default:
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "unknown notification type"})
		return
	}
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
