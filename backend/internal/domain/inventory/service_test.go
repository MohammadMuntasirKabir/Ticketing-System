package inventory

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
	"github.com/you/airline-ticketing/internal/pkg/store"
)

type fakeCap struct{ capacity int }

func (f *fakeCap) FareClassCapacity(ctx context.Context, flightInstanceID, fareClassID int64) (int, error) {
	return f.capacity, nil
}

func newTestService() (*Service, *store.InMemory, *fakeCap) {
	kv := store.NewInMemory()
	cap := &fakeCap{capacity: 100}
	return NewService(kv, cap, zap.NewNop()), kv, cap
}

func TestAvailableFull(t *testing.T) {
	s, _, _ := newTestService()
	ctx := context.Background()
	avail, err := s.Available(ctx, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if avail != 100 {
		t.Fatalf("available = %d, want 100", avail)
	}
}

func TestHoldSeatsReducesAvailability(t *testing.T) {
	s, _, _ := newTestService()
	ctx := context.Background()
	if err := s.HoldSeats(ctx, HoldRequest{FlightInstanceID: 1, FareClassID: 1, Passengers: 3, PNRCode: "ABC123"}); err != nil {
		t.Fatal(err)
	}
	avail, _ := s.Available(ctx, 1, 1)
	if avail != 97 {
		t.Fatalf("available = %d, want 97", avail)
	}
}

func TestHoldSeatsOverCapacity(t *testing.T) {
	s, _, _ := newTestService()
	ctx := context.Background()
	err := s.HoldSeats(ctx, HoldRequest{FlightInstanceID: 1, FareClassID: 1, Passengers: 101, PNRCode: "TOOMANY"})
	if !errors.Is(err, domainErrors.ErrInventoryConflict) {
		t.Fatalf("expected ErrInventoryConflict, got %v", err)
	}
}

func TestHoldSeatsContention(t *testing.T) {
	s, _, _ := newTestService()
	ctx := context.Background()
	// Acquire the lock manually to simulate an in-flight hold.
	if _, err := s.kv.SetNX(ctx, lockKey(1, 1), "1", s.lockTTL); err != nil {
		t.Fatal(err)
	}
	err := s.HoldSeats(ctx, HoldRequest{FlightInstanceID: 1, FareClassID: 1, Passengers: 1, PNRCode: "LOCKED"})
	if !errors.Is(err, domainErrors.ErrInventoryContention) {
		t.Fatalf("expected ErrInventoryContention, got %v", err)
	}
}

func TestHoldSeatsZeroPassengers(t *testing.T) {
	s, _, _ := newTestService()
	ctx := context.Background()
	if err := s.HoldSeats(ctx, HoldRequest{FlightInstanceID: 1, FareClassID: 1, Passengers: 0, PNRCode: "ZERO"}); err == nil {
		t.Fatal("zero passengers should fail")
	}
}

func TestReleaseHolds(t *testing.T) {
	s, _, _ := newTestService()
	ctx := context.Background()
	if err := s.HoldSeats(ctx, HoldRequest{FlightInstanceID: 1, FareClassID: 1, Passengers: 5, PNRCode: "REL"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReleaseHolds(ctx, "REL"); err != nil {
		t.Fatal(err)
	}
	avail, _ := s.Available(ctx, 1, 1)
	if avail != 100 {
		t.Fatalf("available after release = %d, want 100", avail)
	}
}

func TestConfirmHoldsKeepsAvailabilityConsumed(t *testing.T) {
	s, _, _ := newTestService()
	ctx := context.Background()
	if err := s.HoldSeats(ctx, HoldRequest{FlightInstanceID: 1, FareClassID: 1, Passengers: 5, PNRCode: "CONF"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmHolds(ctx, "CONF"); err != nil {
		t.Fatal(err)
	}
	avail, _ := s.Available(ctx, 1, 1)
	if avail != 95 {
		t.Fatalf("available after confirm = %d, want 95", avail)
	}
}
