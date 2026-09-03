package bookingrepo

import (
	"context"
	"testing"
	"time"

	"github.com/you/airline-ticketing/internal/domain/booking"
)

func pnr(code string) *booking.PNR {
	return &booking.PNR{Code: code, UserID: 1, Status: booking.PNRPending, TotalAmount: 100, Currency: "USD"}
}

func TestPNRCRUD(t *testing.T) {
	ctx := context.Background()
	m := New()

	id, err := m.CreatePNR(ctx, pnr("ABC123"))
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("id should be non-zero")
	}

	got, _ := m.GetPNRByCode(ctx, "ABC123")
	if got == nil || got.ID != id {
		t.Fatalf("got: %+v", got)
	}
	byID, _ := m.GetPNR(ctx, id)
	if byID == nil || byID.Code != "ABC123" {
		t.Fatalf("by id: %+v", byID)
	}

	// Duplicate PNR codes conflict.
	if _, err := m.CreatePNR(ctx, pnr("ABC123")); err == nil {
		t.Fatal("duplicate code should fail")
	}

	if err := m.UpdatePNRStatus(ctx, id, booking.PNRConfirmed); err != nil {
		t.Fatal(err)
	}
	byID, _ = m.GetPNR(ctx, id)
	if byID.Status != booking.PNRConfirmed {
		t.Fatalf("status = %q", byID.Status)
	}
}

func TestBookingsAndPassengers(t *testing.T) {
	ctx := context.Background()
	m := New()
	pnrID, _ := m.CreatePNR(ctx, pnr("XYZ999"))

	b1, err := m.CreateBooking(ctx, &booking.Booking{
		PNRID:            pnrID,
		FlightInstanceID: 1000,
		FareClassID:      3,
		Status:           booking.BookingPending,
		TotalPrice:       100,
		Currency:         "USD",
		CreatedAt:        time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := m.CreateBooking(ctx, &booking.Booking{
		PNRID:            pnrID,
		FlightInstanceID: 1000,
		FareClassID:      3,
		Status:           booking.BookingPending,
		TotalPrice:       100,
		Currency:         "USD",
		CreatedAt:        time.Now().UTC(),
	})

	bs, _ := m.GetBookingsByPNR(ctx, pnrID)
	if len(bs) != 2 {
		t.Fatalf("bookings = %d", len(bs))
	}

	// Seat assignment.
	seatID := int64(1201)
	if err := m.UpdateBookingSeat(ctx, b1, &seatID); err != nil {
		t.Fatal(err)
	}
	got, _ := m.GetBooking(ctx, b1)
	if got.SeatID == nil || *got.SeatID != 1201 {
		t.Fatalf("seat: %+v", got.SeatID)
	}

	// Passenger round-trip.
	m.AddPassengers(b2, booking.Passenger{Title: "Ms", FirstName: "Jane", LastName: "Roe"})
	ps, _ := m.GetPassengersByBooking(ctx, b2)
	if len(ps) != 1 || ps[0].FirstName != "Jane" {
		t.Fatalf("passengers: %+v", ps)
	}

	// Status update + list.
	if err := m.UpdateBookingStatus(ctx, b1, booking.BookingConfirmed); err != nil {
		t.Fatal(err)
	}
	all, _ := m.ListBookings(ctx, 0, 0)
	if len(all) != 2 {
		t.Fatalf("list = %d", len(all))
	}
}

func TestDeletePNRCascades(t *testing.T) {
	ctx := context.Background()
	m := New()
	pnrID, _ := m.CreatePNR(ctx, pnr("DEL001"))
	bid, _ := m.CreateBooking(ctx, &booking.Booking{PNRID: pnrID, FlightInstanceID: 1, Status: booking.BookingPending})
	m.AddPassengers(bid, booking.Passenger{Title: "Mr", FirstName: "X"})

	if err := m.DeletePNR(ctx, pnrID); err != nil {
		t.Fatal(err)
	}
	if got, _ := m.GetPNRByCode(ctx, "DEL001"); got != nil {
		t.Fatal("pnr should be gone")
	}
	if got, _ := m.GetBooking(ctx, bid); got != nil {
		t.Fatal("booking should cascade-delete")
	}
	if ps, _ := m.GetPassengersByBooking(ctx, bid); len(ps) != 0 {
		t.Fatal("passengers should cascade-delete")
	}
}

func TestMissingEntities(t *testing.T) {
	ctx := context.Background()
	m := New()
	if got, _ := m.GetPNRByCode(ctx, "NOPE"); got != nil {
		t.Fatal("missing pnr should be nil")
	}
	if got, _ := m.GetPNR(ctx, 999); got != nil {
		t.Fatal("missing pnr should be nil")
	}
	if got, _ := m.GetBooking(ctx, 999); got != nil {
		t.Fatal("missing booking should be nil")
	}
	if err := m.UpdatePNRStatus(ctx, 999, booking.PNRConfirmed); err == nil {
		t.Fatal("update of missing pnr should fail")
	}
	if err := m.DeletePNR(ctx, 999); err == nil {
		t.Fatal("delete of missing pnr should fail")
	}
}
