package handler

import (
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/api/helper"
	"github.com/you/airline-ticketing/internal/domain/payment"
)

type PaymentHandler struct {
	provider payment.Provider
	webhook  payment.WebhookVerifier
	log      *zap.Logger
}

func NewPaymentHandler(provider payment.Provider, webhook payment.WebhookVerifier, log *zap.Logger) *PaymentHandler {
	return &PaymentHandler{provider: provider, webhook: webhook, log: log}
}

type createIntentRequest struct {
	PNRID    int64   `json:"pnr_id"`
	Amount   float64 `json:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" validate:"required,len=3"`
}

type createIntentResponse struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	AmountCents  int64  `json:"amount_cents"`
	Currency     string `json:"currency"`
	Status       string `json:"status"`
}

func (h *PaymentHandler) CreateIntent(w http.ResponseWriter, r *http.Request) {
	var in createIntentRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	pi, err := h.provider.CreatePaymentIntent(r.Context(), in.PNRID, in.Amount, in.Currency)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, createIntentResponse{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		AmountCents:  pi.Amount,
		Currency:     pi.Currency,
		Status:       pi.Status,
	})
}

func (h *PaymentHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	if h.webhook == nil {
		helper.JSON(w, http.StatusNotImplemented, map[string]string{"error": "webhook verification is not configured"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read request body"})
		return
	}
	evt, err := h.webhook.ParseWebhook(r, r.Header.Get("Stripe-Signature"), body)
	if err != nil {
		helper.Error(w, err)
		return
	}
	h.log.Info("stripe webhook received",
		zap.String("type", evt.Type),
		zap.String("payment_id", evt.PaymentID),
		zap.String("pnr_id", evt.PNRID))
	helper.JSON(w, http.StatusOK, map[string]string{"received": "true"})
}

type refundRequest struct {
	PaymentIntentID string   `json:"payment_intent_id" validate:"required"`
	Amount          *float64 `json:"amount"`
}

type refundResponse struct {
	ID     string `json:"id"`
	Amount int64  `json:"amount"`
	Status string `json:"status"`
}

func (h *PaymentHandler) Refund(w http.ResponseWriter, r *http.Request) {
	var in refundRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	ref, err := h.provider.Refund(r.Context(), in.PaymentIntentID, in.Amount)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, refundResponse{ID: ref.ID, Amount: ref.Amount, Status: ref.Status})
}
