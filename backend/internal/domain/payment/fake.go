package payment

import (
	"context"
	"sync"

	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

// FakeProvider is an in-memory Provider implementation used by tests that
// need to exercise the booking flow without hitting Stripe.
type FakeProvider struct {
	mu       sync.Mutex
	intents  map[int64]*PaymentIntentResult
	statuses map[string]string // paymentIntentID -> status
	refunds  []RefundResult
}

func NewFake() *FakeProvider {
	return &FakeProvider{
		intents:  map[int64]*PaymentIntentResult{},
		statuses: map[string]string{},
	}
}

func (f *FakeProvider) CreatePaymentIntent(ctx context.Context, pnrID int64, amount float64, currency string) (*PaymentIntentResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	pi := &PaymentIntentResult{
		ID:           "pi_fake_" + itoa(pnrID),
		ClientSecret: "pi_fake_secret_" + itoa(pnrID),
		Amount:       toCents(amount),
		Currency:     currency,
		Status:       "requires_payment_method",
	}
	f.intents[pnrID] = pi
	f.statuses[pi.ID] = "requires_payment_method"
	return pi, nil
}

func (f *FakeProvider) VerifyPayment(ctx context.Context, paymentIntentID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if st, ok := f.statuses[paymentIntentID]; ok {
		return st, nil
	}
	return "", domainErrors.ErrPaymentNotConfirmed
}

// SetStatus lets tests simulate a webhook-driven status change.
func (f *FakeProvider) SetStatus(paymentIntentID, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statuses[paymentIntentID] = status
}

func (f *FakeProvider) Refund(ctx context.Context, paymentIntentID string, amount *float64) (*RefundResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.statuses[paymentIntentID]; !ok {
		return nil, domainErrors.ErrPaymentNotConfirmed
	}
	r := RefundResult{ID: "re_fake", Amount: toCents(derefAmount(amount)), Status: "succeeded"}
	f.refunds = append(f.refunds, r)
	return &r, nil
}

func derefAmount(a *float64) float64 {
	if a == nil {
		return 0
	}
	return *a
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
