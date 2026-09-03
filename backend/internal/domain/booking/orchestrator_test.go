package booking_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/domain/booking"
	"github.com/you/airline-ticketing/internal/domain/inventory"
	"github.com/you/airline-ticketing/internal/domain/payment"
	"github.com/you/airline-ticketing/internal/domain/pricing"
	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
	"github.com/you/airline-ticketing/internal/pkg/store"
	"github.com/you/airline-ticketing/internal/repository/bookingrepo"
	"github.com/you/airline-ticketing/internal/repository/flightrepo"
)

type stubNotifier struct{ sent int }

func (s *stubNotifier) SendBookingConfirmation(ctx context.Context, pnrID int64) error {
	s.sent++
	return nil
}

func newOrchestrator(t *testing.T) (*booking.Orchestrator, *flightrepo.Memory, *payment.FakeProvider, *stubNotifier, store.KV) {
	t.Helper()
	logger := zap.NewNop()
	kv := store.NewInMemory()
	flights := flightrepo.New()
	inv := inventory.NewService(kv, flights, logger)
	price := pricing.NewEngine(logger)
	pay := payment.NewFake()
	bookings := bookingrepo.New()
	notify := &stubNotifier{}
	orch := booking.NewOrchestrator(flights, inv, price, pay, bookings, kv, notify, logger)
	return orch, flights, pay, notify, kv
}

func holdCount(t *testing.T, kv store.KV) int {
	t.Helper()
	keys, err := kv.Keys(context.Background(), "inv:hold:*:*:*")
	if err != nil {
		t.Fatal(err)
	}
	return len(keys)
}

func passenger(n int) []booking.PassengerInput {
	ps := make([]booking.PassengerInput, n)
	for i := range ps {
		ps[i] = booking.PassengerInput{
			Title:          "Mr",
			FirstName:      "John",
			LastName:       "Doe",
			DateOfBirth:    "1990-01-01",
			Nationality:    "US",
			PassportNumber: "X1234567",
			PassportExpiry: "2030-01-01",
		}
	}
	return ps
}

func TestQuoteCreateConfirm(t *testing.T) {
	orch, flights, pay, notify, kv := newOrchestrator(t)
	ctx := context.Background()

	quote, err := orch.GetPriceQuote(ctx, booking.PriceQuoteRequest{
		FlightInstanceID: 1000,
		FareClassCode:    "YSAVER",
		Passengers:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if quote.QuoteID == "" || quote.Total <= 0 || quote.FlightID != 1000 {
		t.Fatalf("quote: %+v", quote)
	}

	resp, err := orch.CreateBooking(ctx, booking.CreateBookingRequest{
		QuoteID:        quote.QuoteID,
		IdempotencyKey: "e2e-1",
		Passengers:     passenger(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.PNR == "" || resp.PNRID == 0 || resp.PaymentIntentID == "" || resp.AmountCents <= 0 {
		t.Fatalf("create response: %+v", resp)
	}
	if resp.Status != string(booking.PNRPending) {
		t.Fatalf("status = %q", resp.Status)
	}
	if notify.sent != 1 {
		t.Fatalf("confirmation notifications = %d, want 1", notify.sent)
	}

	// The hold reduced available inventory for the fare class.
	avail, err := flights.FareClassCapacity(ctx, 1000, 3)
	if err != nil {
		t.Fatal(err)
	}
	holds := holdCount(t, kv)
	if holds != 1 {
		t.Fatalf("holds = %d, want 1", holds)
	}
	if avail < 1 {
		t.Fatal("capacity should still be >= 1")
	}

	// Simulate the payment webhook firing, then confirm.
	pay.SetStatus(resp.PaymentIntentID, "succeeded")
	confirmed, err := orch.ConfirmBooking(ctx, resp.PNR)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Status != string(booking.PNRConfirmed) {
		t.Fatalf("confirmed status = %q", confirmed.Status)
	}

	detail, err := orch.GetByPNR(ctx, resp.PNR)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Status != string(booking.PNRConfirmed) || len(detail.Bookings) != 1 {
		t.Fatalf("detail: %+v", detail)
	}
}

func TestIdempotencyReplay(t *testing.T) {
	orch, _, pay, _, kv := newOrchestrator(t)
	ctx := context.Background()

	quote, _ := orch.GetPriceQuote(ctx, booking.PriceQuoteRequest{FlightInstanceID: 1000, FareClassCode: "YSAVER", Passengers: 2})
	req := booking.CreateBookingRequest{QuoteID: quote.QuoteID, IdempotencyKey: "e2e-idem", Passengers: passenger(2)}

	first, err := orch.CreateBooking(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	pay.SetStatus(first.PaymentIntentID, "succeeded")
	if _, err := orch.ConfirmBooking(ctx, first.PNR); err != nil {
		t.Fatal(err)
	}

	// Replay with the same key returns the cached result (same PNR).
	replayed, err := orch.CreateBooking(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.PNR != first.PNR {
		t.Fatalf("replay pnr = %q, want %q", replayed.PNR, first.PNR)
	}
	// No extra hold was created: the confirmed booking's hold still
	// consumes capacity until cancelled, so the count stays at 1.
	holds := holdCount(t, kv)
	if holds != 1 {
		t.Fatalf("holds after replay = %d, want 1", holds)
	}
}

func TestQuoteExpiry(t *testing.T) {
	orch, _, _, _, kv := newOrchestrator(t)
	ctx := context.Background()

	quote, _ := orch.GetPriceQuote(ctx, booking.PriceQuoteRequest{FlightInstanceID: 1000, FareClassCode: "YSAVER", Passengers: 1})

	// Overwrite the cached quote with an already-expired one.
	expired := booking.PriceQuote{
		FlightInstanceID: 1000,
		FareClassCode:    "YSAVER",
		Passengers:       1,
		TotalAmount:      189,
		ExpiresAt:        time.Now().Add(-time.Hour),
	}
	data, _ := json.Marshal(expired)
	_ = kv.Set(ctx, "quote:"+quote.QuoteID, string(data), time.Minute)

	_, err := orch.CreateBooking(ctx, booking.CreateBookingRequest{QuoteID: quote.QuoteID, Passengers: passenger(1)})
	if !errors.Is(err, domainErrors.ErrQuoteExpired) {
		t.Fatalf("expected ErrQuoteExpired, got %v", err)
	}
}

func TestPassengerCountMismatch(t *testing.T) {
	orch, _, _, _, _ := newOrchestrator(t)
	ctx := context.Background()

	quote, _ := orch.GetPriceQuote(ctx, booking.PriceQuoteRequest{FlightInstanceID: 1000, FareClassCode: "YSAVER", Passengers: 2})
	_, err := orch.CreateBooking(ctx, booking.CreateBookingRequest{QuoteID: quote.QuoteID, Passengers: passenger(1)})
	if err == nil {
		t.Fatal("expected passenger-count mismatch error")
	}
}

func TestCancelReleasesHold(t *testing.T) {
	orch, _, pay, _, kv := newOrchestrator(t)
	ctx := context.Background()

	quote, _ := orch.GetPriceQuote(ctx, booking.PriceQuoteRequest{FlightInstanceID: 1000, FareClassCode: "YSAVER", Passengers: 1})
	resp, err := orch.CreateBooking(ctx, booking.CreateBookingRequest{QuoteID: quote.QuoteID, Passengers: passenger(1)})
	if err != nil {
		t.Fatal(err)
	}
	holds := holdCount(t, kv)
	if holds != 1 {
		t.Fatalf("holds = %d, want 1", holds)
	}

	pay.SetStatus(resp.PaymentIntentID, "succeeded")
	if _, err := orch.ConfirmBooking(ctx, resp.PNR); err != nil {
		t.Fatal(err)
	}

	cancelled, err := orch.CancelBooking(ctx, resp.PNR)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != string(booking.PNRCancelled) {
		t.Fatalf("cancel status = %q", cancelled.Status)
	}
	holds = holdCount(t, kv)
	if holds != 0 {
		t.Fatalf("holds after cancel = %d, want 0", holds)
	}
}
