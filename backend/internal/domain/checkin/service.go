package checkin

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/domain/booking"
	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
	"github.com/you/airline-ticketing/internal/pkg/seat"
)

// FlightInfoProvider resolves the human-readable flight details needed on
// a boarding pass. The flight repository satisfies this interface.
type FlightInfoProvider interface {
	FlightInfo(ctx context.Context, flightInstanceID int64) (flightNumber, origin, destination string, err error)
}

// Service manages the check-in use-case: initiate check-in, assign a seat
// and issue a boarding pass. It is backed by the booking repository for
// passenger/booking data and a flight info provider for route details.
type Service struct {
	bookings booking.Repository
	flights  FlightInfoProvider
	logger   *zap.Logger
}

func NewService(bookings booking.Repository, flights FlightInfoProvider, logger *zap.Logger) *Service {
	return &Service{bookings: bookings, flights: flights, logger: logger}
}

// InitiateCheckIn marks a PNR as checked-in and returns the check-in
// record spanning all passengers on the PNR.
func (s *Service) InitiateCheckIn(ctx context.Context, pnrCode string) (*CheckIn, error) {
	pnr, err := s.bookings.GetPNRByCode(ctx, pnrCode)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load pnr", err)
	}
	if pnr == nil {
		return nil, domainErrors.ErrPNRNotFound
	}
	if pnr.Status == booking.PNRCancelled || pnr.Status == booking.PNRRefunded {
		return nil, domainErrors.Validation("cannot check in a cancelled/refunded booking")
	}
	if pnr.Status == booking.PNRCheckedIn {
		return nil, domainErrors.ErrBookingAlreadyConfirmed
	}

	bs, err := s.bookings.GetBookingsByPNR(ctx, pnr.ID)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load bookings", err)
	}
	if len(bs) == 0 {
		return nil, domainErrors.NotFound("no bookings on this pnr")
	}

	now := time.Now().UTC()
	_ = s.bookings.UpdatePNRStatus(ctx, pnr.ID, booking.PNRCheckedIn)
	return &CheckIn{
		ID:           bs[0].ID,
		PNRID:        pnr.ID,
		BookingID:    bs[0].ID,
		CheckedInAt:  now,
		IsOnline:     true,
		BaggageCount: 0,
	}, nil
}

// AssignSeat assigns a seat to a booking on the PNR and returns the
// boarding pass data for that passenger.
func (s *Service) AssignSeat(ctx context.Context, pnrCode, seatNumber string) (*BoardingPass, error) {
	if seatNumber == "" {
		return nil, domainErrors.Validation("seat number is required")
	}
	pnr, err := s.bookings.GetPNRByCode(ctx, pnrCode)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load pnr", err)
	}
	if pnr == nil {
		return nil, domainErrors.ErrPNRNotFound
	}
	bs, err := s.bookings.GetBookingsByPNR(ctx, pnr.ID)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load bookings", err)
	}
	if len(bs) == 0 {
		return nil, domainErrors.Validation("no bookings to assign a seat")
	}

	seatID := seatNumberToID(seatNumber)
	if seatID == nil {
		return nil, domainErrors.Validation("invalid seat number")
	}
	if err := s.bookings.UpdateBookingSeat(ctx, bs[0].ID, seatID); err != nil {
		return nil, domainErrors.InternalWrap("failed to assign seat", err)
	}

	bp, err := s.buildBoardingPass(ctx, pnr.Code, &bs[0], seatNumber)
	if err != nil {
		return nil, err
	}
	s.logger.Info("seat assigned",
		zap.String("pnr", pnrCode),
		zap.String("seat", seatNumber),
		zap.Int64("booking_id", bs[0].ID))
	return bp, nil
}

// GetBoardingPass returns the boarding pass for the first booking on a PNR.
func (s *Service) GetBoardingPass(ctx context.Context, pnrCode string) (*BoardingPass, error) {
	pnr, err := s.bookings.GetPNRByCode(ctx, pnrCode)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load pnr", err)
	}
	if pnr == nil {
		return nil, domainErrors.ErrPNRNotFound
	}
	bs, err := s.bookings.GetBookingsByPNR(ctx, pnr.ID)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load bookings", err)
	}
	if len(bs) == 0 {
		return nil, domainErrors.NotFound("no bookings on this pnr")
	}

	seatLabel := ""
	if bs[0].SeatID != nil {
		seatLabel = seat.DecodeLabel(*bs[0].SeatID)
	}
	if seatLabel == "" {
		seatLabel = "TBA"
	}
	return s.buildBoardingPass(ctx, pnr.Code, &bs[0], seatLabel)
}

func (s *Service) buildBoardingPass(ctx context.Context, pnrCode string, b *booking.Booking, seat string) (*BoardingPass, error) {
	passengers, err := s.bookings.GetPassengersByBooking(ctx, b.ID)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load passengers", err)
	}
	passengerID := int64(0)
	if len(passengers) > 0 {
		passengerID = passengers[0].ID
	}

	flightNumber, origin, destination := "unknown", "unknown", "unknown"
	fn, o_, d_, ferr := s.flights.FlightInfo(ctx, b.FlightInstanceID)
	if ferr == nil {
		flightNumber, origin, destination = fn, o_, d_
	}

	now := time.Now().UTC()
	return &BoardingPass{
		ID:             b.ID,
		CheckInID:      b.ID,
		PassengerID:    passengerID,
		FlightNumber:   flightNumber,
		Origin:         origin,
		Destination:    destination,
		SeatNumber:     seat,
		BoardingGroup:  "B",
		SequenceNumber: int(b.ID) % 30,
		Gate:           "A12",
		BoardingTime:   now.Add(45 * time.Minute),
		QRCodeData:     fmt.Sprintf("PNR:%s|FLIGHT:%s|SEAT:%s", pnrCode, flightNumber, seat),
		IssuedAt:       now,
	}, nil
}

// seatNumberToID maps a seat label (e.g. "12A") to its stable seat id
// using the shared seat encoding.
func seatNumberToID(s string) *int64 {
	row, col, ok := seat.ParseLabel(s)
	if !ok {
		return nil
	}
	id := seat.EncodeID(row, col)
	return &id
}
