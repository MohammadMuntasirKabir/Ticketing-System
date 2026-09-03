package checkin

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/domain/booking"
	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
	"github.com/you/airline-ticketing/internal/repository/bookingrepo"
)

type fakeFlights struct{}

func (f *fakeFlights) FlightInfo(ctx context.Context, flightInstanceID int64) (string, string, string, error) {
	return "AA101", "JFK", "LAX", nil
}

func newCheckinService() (*Service, *bookingrepo.Memory) {
	repo := bookingrepo.New()
	return NewService(repo, &fakeFlights{}, zap.NewNop()), repo
}

func seedBooking(t *testing.T, repo *bookingrepo.Memory) (*booking.PNR, []booking.Booking) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	exp := now.Add(30 * time.Minute)
	pnr := &booking.PNR{
		Code:        "ABC123",
		UserID:      1,
		Status:      booking.PNRConfirmed,
		TotalAmount: 200,
		Currency:    "USD",
		ExpiresAt:   &exp,
	}
	id, err := repo.CreatePNR(ctx, pnr)
	if err != nil {
		t.Fatal(err)
	}
	pnr.ID = id
	bid, err := repo.CreateBooking(ctx, &booking.Booking{
		PNRID:            id,
		FlightInstanceID: 1000,
		FareClassID:      3,
		Status:           booking.BookingConfirmed,
		TotalPrice:       200,
		Currency:         "USD",
	})
	if err != nil {
		t.Fatal(err)
	}
	repo.AddPassengers(bid, booking.Passenger{
		Title:     "Mr",
		FirstName: "John",
		LastName:  "Doe",
	})
	bs, _ := repo.GetBookingsByPNR(ctx, id)
	return pnr, bs
}

func TestInitiateCheckIn(t *testing.T) {
	s, repo := newCheckinService()
	pnr, _ := seedBooking(t, repo)
	ctx := context.Background()

	ci, err := s.InitiateCheckIn(ctx, pnr.Code)
	if err != nil {
		t.Fatal(err)
	}
	if !ci.IsOnline || ci.PNRID != pnr.ID {
		t.Fatalf("checkin: %+v", ci)
	}

	// Second check-in should fail.
	if _, err := s.InitiateCheckIn(ctx, pnr.Code); !errors.Is(err, domainErrors.ErrBookingAlreadyConfirmed) {
		t.Fatalf("expected already-confirmed error, got %v", err)
	}
}

func TestInitiateCheckInCancelled(t *testing.T) {
	s, repo := newCheckinService()
	pnr, _ := seedBooking(t, repo)
	ctx := context.Background()
	_ = repo.UpdatePNRStatus(ctx, pnr.ID, booking.PNRCancelled)

	if _, err := s.InitiateCheckIn(ctx, pnr.Code); err == nil {
		t.Fatal("check-in of a cancelled booking should fail")
	}
}

func TestInitiateCheckInNotFound(t *testing.T) {
	s, _ := newCheckinService()
	if _, err := s.InitiateCheckIn(context.Background(), "NOPE99"); !errors.Is(err, domainErrors.ErrPNRNotFound) {
		t.Fatalf("expected PNR not found, got %v", err)
	}
}

func TestAssignSeat(t *testing.T) {
	s, repo := newCheckinService()
	pnr, _ := seedBooking(t, repo)
	ctx := context.Background()

	bp, err := s.AssignSeat(ctx, pnr.Code, "12A")
	if err != nil {
		t.Fatal(err)
	}
	if bp.SeatNumber != "12A" {
		t.Fatalf("seat = %q", bp.SeatNumber)
	}
	if bp.FlightNumber != "AA101" || bp.Origin != "JFK" || bp.Destination != "LAX" {
		t.Fatalf("boarding pass: %+v", bp)
	}
	if bp.QRCodeData == "" {
		t.Fatal("qr code should be present")
	}
}

func TestAssignSeatInvalid(t *testing.T) {
	s, repo := newCheckinService()
	pnr, _ := seedBooking(t, repo)
	ctx := context.Background()

	if _, err := s.AssignSeat(ctx, pnr.Code, ""); err == nil {
		t.Fatal("empty seat should fail")
	}
	if _, err := s.AssignSeat(ctx, pnr.Code, "99ZZ"); err == nil {
		t.Fatal("invalid seat label should fail")
	}
}

func TestGetBoardingPassTBA(t *testing.T) {
	s, repo := newCheckinService()
	pnr, _ := seedBooking(t, repo)
	ctx := context.Background()

	bp, err := s.GetBoardingPass(ctx, pnr.Code)
	if err != nil {
		t.Fatal(err)
	}
	if bp.SeatNumber != "TBA" {
		t.Fatalf("seat should be TBA, got %q", bp.SeatNumber)
	}
}
