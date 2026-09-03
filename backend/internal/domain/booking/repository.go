package booking

import "context"

// Repository persists durable booking records (PNRs, bookings,
// passengers). The transient quote/hold state lives in the KV store.
// Implementations: postgres (prod), in-memory (tests).
type Repository interface {
	CreatePNR(ctx context.Context, p *PNR) (int64, error)
	GetPNRByCode(ctx context.Context, code string) (*PNR, error)
	GetPNR(ctx context.Context, id int64) (*PNR, error)
	UpdatePNRStatus(ctx context.Context, pnrID int64, status PNRStatus) error
	DeletePNR(ctx context.Context, pnrID int64) error

	CreateBooking(ctx context.Context, b *Booking) (int64, error)
	GetBookingsByPNR(ctx context.Context, pnrID int64) ([]Booking, error)
	GetBooking(ctx context.Context, id int64) (*Booking, error)
	UpdateBookingStatus(ctx context.Context, id int64, status BookingStatus) error
	UpdateBookingSeat(ctx context.Context, bookingID int64, seatID *int64) error
	GetPassengersByBooking(ctx context.Context, bookingID int64) ([]Passenger, error)
	ListBookings(ctx context.Context, limit, offset int) ([]Booking, error)
}

// FareClassProvider resolves a fare-class code to its stable ID and
// capacity for a flight instance. The flight repository satisfies it.
type FareClassProvider interface {
	ResolveFareClass(ctx context.Context, flightInstanceID int64, code string) (fareClassID int64, capacity int, err error)
}

// PassengerCounter counts confirmed passengers for capacity reporting.
type PassengerCounter interface {
	CountConfirmedPassengers(ctx context.Context, flightInstanceID int64) (int, error)
}
