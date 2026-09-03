package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

func TestFakeProviderLifecycle(t *testing.T) {
	ctx := context.Background()
	f := NewFake()

	pi, err := f.CreatePaymentIntent(ctx, 42, 100.00, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if pi.ID == "" || pi.ClientSecret == "" || pi.Amount != 10000 {
		t.Fatalf("payment intent: %+v", pi)
	}

	st, err := f.VerifyPayment(ctx, pi.ID)
	if err != nil || st != "requires_payment_method" {
		t.Fatalf("verify = %q, %v", st, err)
	}

	// Simulate a webhook flipping the status to succeeded.
	f.SetStatus(pi.ID, "succeeded")
	st, err = f.VerifyPayment(ctx, pi.ID)
	if err != nil || st != "succeeded" {
		t.Fatalf("verify after webhook = %q, %v", st, err)
	}

	r, err := f.Refund(ctx, pi.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "succeeded" {
		t.Fatalf("refund: %+v", r)
	}
}

func TestFakeProviderUnknownIntent(t *testing.T) {
	ctx := context.Background()
	f := NewFake()
	if _, err := f.VerifyPayment(ctx, "pi_nope"); !isErr(err, domainErrors.ErrPaymentNotConfirmed) {
		t.Fatalf("expected ErrPaymentNotConfirmed, got %v", err)
	}
	if _, err := f.Refund(ctx, "pi_nope", nil); !isErr(err, domainErrors.ErrPaymentNotConfirmed) {
		t.Fatalf("expected ErrPaymentNotConfirmed, got %v", err)
	}
}

// signedWebhookHeader builds a Stripe-style `t=...;v1=...` header that
// verifies under the given secret.
func signedWebhookHeader(secret string, body []byte) string {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	signed := ts + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	sig := hex.EncodeToString(mac.Sum(nil))
	return "t=" + ts + ",v1=" + sig
}

func TestParseWebhookValid(t *testing.T) {
	secret := "whsec_test"
	p := NewStripeProvider("sk_test", secret, zap.NewNop())

	body := `{"id":"evt_1","type":"payment_intent.succeeded","data":{"object":{"id":"pi_123","amount":10000,"currency":"usd","metadata":{"pnr_id":"77"},"status":"succeeded"}}}`
	req, _ := http.NewRequest("POST", "/payments/webhook", strings.NewReader(body))
	evt, err := p.ParseWebhook(req, signedWebhookHeader(secret, []byte(body)), []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if evt.Type != "payment_intent.succeeded" {
		t.Fatalf("type = %q", evt.Type)
	}
	if evt.PaymentID != "pi_123" || evt.Amount != 10000 || evt.PNRID != "77" {
		t.Fatalf("event: %+v", evt)
	}
}

func TestParseWebhookBadSignature(t *testing.T) {
	secret := "whsec_test"
	p := NewStripeProvider("sk_test", secret, zap.NewNop())

	body := []byte(`{"type":"payment_intent.succeeded","data":{}}`)
	req, _ := http.NewRequest("POST", "/payments/webhook", strings.NewReader(string(body)))
	if _, err := p.ParseWebhook(req, "t=1,v1=deadbeef", body); !isErr(err, domainErrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParseWebhookTamperedBody(t *testing.T) {
	secret := "whsec_test"
	p := NewStripeProvider("sk_test", secret, zap.NewNop())

	validBody := []byte(`{"type":"payment_intent.succeeded","data":{"object":{"id":"pi_1"}}}`)
	header := signedWebhookHeader(secret, validBody)

	// Sign one body but send another: signature must not verify.
	tampered := []byte(`{"type":"payment_intent.payment_failed","data":{"object":{"id":"pi_1"}}}`)
	req, _ := http.NewRequest("POST", "/payments/webhook", strings.NewReader(string(tampered)))
	if _, err := p.ParseWebhook(req, header, tampered); !isErr(err, domainErrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParseWebhookRefunded(t *testing.T) {
	secret := "whsec_test"
	p := NewStripeProvider("sk_test", secret, zap.NewNop())

	body := []byte(`{"type":"charge.refunded","data":{"object":{"id":"pi_9","refunds":{"data":[{"amount":5000}]}}}}`)
	req, _ := http.NewRequest("POST", "/payments/webhook", strings.NewReader(string(body)))
	evt, err := p.ParseWebhook(req, signedWebhookHeader(secret, body), body)
	if err != nil {
		t.Fatal(err)
	}
	if evt.Type != "charge.refunded" || evt.RefundAmount != 5000 {
		t.Fatalf("event: %+v", evt)
	}
}

func TestParseWebhookStaleTimestamp(t *testing.T) {
	secret := "whsec_test"
	p := NewStripeProvider("sk_test", secret, zap.NewNop())

	body := []byte(`{"type":"payment_intent.succeeded","data":{}}`)
	// Timestamp 10 minutes in the past exceeds the 300s tolerance.
	ts := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
	signed := ts + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	header := "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))

	req, _ := http.NewRequest("POST", "/payments/webhook", strings.NewReader(string(body)))
	if _, err := p.ParseWebhook(req, header, body); !isErr(err, domainErrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for stale timestamp, got %v", err)
	}
}

func isErr(err error, want error) bool {
	if err == nil {
		return false
	}
	return domainErrors.HTTPStatus(err) == domainErrors.HTTPStatus(want) || err.Error() == want.Error()
}
