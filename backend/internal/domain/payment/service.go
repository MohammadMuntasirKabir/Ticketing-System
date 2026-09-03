package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/paymentintent"
	"github.com/stripe/stripe-go/v86/refund"
	"go.uber.org/zap"

	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

// Provider abstracts the payment gateway. A Stripe-backed implementation
// is used in production; in-memory fakes back the tests.
type Provider interface {
	CreatePaymentIntent(ctx context.Context, pnrID int64, amount float64, currency string) (*PaymentIntentResult, error)
	VerifyPayment(ctx context.Context, paymentIntentID string) (string, error)
	Refund(ctx context.Context, paymentIntentID string, amount *float64) (*RefundResult, error)
}

// WebhookVerifier validates a webhook signature and decodes the event.
// The Stripe-backed provider implements it.
type WebhookVerifier interface {
	ParseWebhook(r *http.Request, signatureHeader string, body []byte) (*WebhookEvent, error)
}

type PaymentIntentResult struct {
	ID           string
	ClientSecret string
	Amount       int64
	Currency     string
	Status       string
}

type RefundResult struct {
	ID     string
	Amount int64
	Status string
}

// WebhookEvent is the domain representation of a Stripe webhook event,
// decoupled from the stripe SDK types.
type WebhookEvent struct {
	Type         string
	PaymentID    string
	Amount       int64
	Currency     string
	PNRID        string
	RefundAmount int64
}

type StripeProvider struct {
	webhookSecret string
	logger        *zap.Logger
}

func NewStripeProvider(secretKey, webhookSecret string, logger *zap.Logger) *StripeProvider {
	stripe.Key = secretKey
	return &StripeProvider{webhookSecret: webhookSecret, logger: logger}
}

func (p *StripeProvider) CreatePaymentIntent(ctx context.Context, pnrID int64, amount float64, currency string) (*PaymentIntentResult, error) {
	amountCents := toCents(amount)
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(currency),
		Metadata: map[string]string{
			"pnr_id": fmt.Sprintf("%d", pnrID),
		},
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		p.logger.Error("stripe: failed to create payment intent", zap.Error(err))
		return nil, domainErrors.InternalWrap("payment provider error", err)
	}
	return &PaymentIntentResult{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Amount:       amountCents,
		Currency:     currency,
		Status:       string(pi.Status),
	}, nil
}

func (p *StripeProvider) VerifyPayment(ctx context.Context, paymentIntentID string) (string, error) {
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		return "", domainErrors.ErrPaymentNotConfirmed
	}
	return string(pi.Status), nil
}

func (p *StripeProvider) Refund(ctx context.Context, paymentIntentID string, amount *float64) (*RefundResult, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
	}
	if amount != nil {
		params.Amount = stripe.Int64(toCents(*amount))
	}
	r, err := refund.New(params)
	if err != nil {
		p.logger.Error("stripe: failed to create refund", zap.Error(err))
		return nil, domainErrors.ErrPaymentFailed
	}
	return &RefundResult{
		ID:     r.ID,
		Amount: r.Amount,
		Status: string(r.Status),
	}, nil
}

// ParseWebhook validates the Stripe signature and decodes the event into a
// domain WebhookEvent. Returns ErrInvalidToken on any signature failure.
func (p *StripeProvider) ParseWebhook(r *http.Request, signatureHeader string, body []byte) (*WebhookEvent, error) {
	const maxBodyBytes = 65536
	if int64(len(body)) > maxBodyBytes {
		return nil, domainErrors.Validation("webhook payload too large")
	}

	// Verify the signature using the secret directly (constant-time).
	if !p.verifySignature(body, signatureHeader) {
		return nil, domainErrors.ErrInvalidToken
	}

	var raw struct {
		Type   string          `json:"type"`
		Data   json.RawMessage `json:"data"`
		APIVer string          `json:"api_version"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, domainErrors.Validation("invalid webhook JSON")
	}

	evt := &WebhookEvent{Type: raw.Type}

	// Unmarshal the nested object based on event type.
	if raw.Type == "payment_intent.succeeded" || raw.Type == "payment_intent.payment_failed" {
		var obj struct {
			ID       string            `json:"id"`
			Amount   int64             `json:"amount"`
			Currency string            `json:"currency"`
			Metadata map[string]string `json:"metadata"`
			Status   string            `json:"status"`
		}
		_ = json.Unmarshal(raw.Data, &obj)
		// The nested object is under data.object; handle both shapes.
		var nested struct {
			Object struct {
				ID       string            `json:"id"`
				Amount   int64             `json:"amount"`
				Currency string            `json:"currency"`
				Metadata map[string]string `json:"metadata"`
				Status   string            `json:"status"`
			} `json:"object"`
		}
		if err := json.Unmarshal(raw.Data, &nested); err == nil {
			evt.PaymentID = nested.Object.ID
			evt.Amount = nested.Object.Amount
			evt.Currency = nested.Object.Currency
			if v, ok := nested.Object.Metadata["pnr_id"]; ok {
				evt.PNRID = v
			}
		} else {
			evt.PaymentID = obj.ID
			evt.Amount = obj.Amount
			evt.Currency = obj.Currency
			if v, ok := obj.Metadata["pnr_id"]; ok {
				evt.PNRID = v
			}
		}
	} else if raw.Type == "charge.refunded" {
		var nested struct {
			Object struct {
				ID      string `json:"id"`
				Amount  int64  `json:"amount"`
				Refunds struct {
					Data []struct {
						Amount int64 `json:"amount"`
					} `json:"data"`
				} `json:"refunds"`
			} `json:"object"`
		}
		if err := json.Unmarshal(raw.Data, &nested); err == nil {
			evt.PaymentID = nested.Object.ID
			evt.RefundAmount = nested.Object.Refunds.Data[0].Amount
		}
	}

	return evt, nil
}

// verifySignature reimplements Stripe's v1 signature check so it can be
// unit-tested offline without network access. The header format is:
//
//	t=<timestamp>,v1=<hex hmac>
func (p *StripeProvider) verifySignature(payload []byte, header string) bool {
	ts, sig, ok := parseSignatureHeader(header)
	if !ok {
		return false
	}
	signed := fmt.Sprintf("%s.%s", ts, string(payload))
	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write([]byte(signed))
	expected := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison of the computed HMAC.
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return false
	}
	var t int64
	if _, err := fmt.Sscanf(ts, "%d", &t); err != nil {
		return false
	}
	const tolerance = 300
	now := time.Now().Unix()
	return absDiff(now, t) <= tolerance
}

func parseSignatureHeader(header string) (ts string, sig string, ok bool) {
	for _, p := range strings.Split(header, ",") {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "v1":
			sig = kv[1]
		}
	}
	return ts, sig, ts != "" && sig != ""
}

func toCents(amount float64) int64 {
	return int64(amount*100 + 0.5)
}

func absDiff(a, b int64) int64 {
	if a > b {
		return a - b
	}
	return b - a
}
