package booking

import (
	"time"
)

type PNRStatus string

const (
	PNRPending   PNRStatus = "PENDING"
	PNRConfirmed PNRStatus = "CONFIRMED"
	PNRCancelled PNRStatus = "CANCELLED"
	PNRCheckedIn PNRStatus = "CHECKED_IN"
	PNRBoarded   PNRStatus = "BOARDED"
	PNRNoShow    PNRStatus = "NO_SHOW"
	PNRRefunded  PNRStatus = "REFUNDED"
)

type BookingStatus string

const (
	BookingPending   BookingStatus = "PENDING"
	BookingConfirmed BookingStatus = "CONFIRMED"
	BookingCancelled BookingStatus = "CANCELLED"
)

type PNR struct {
	ID          int64
	Code        string
	UserID      int64
	Status      PNRStatus
	TotalAmount float64
	Currency    string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Booking struct {
	ID               int64
	PNRID            int64
	FlightInstanceID int64
	FareClassID      int64
	SeatID           *int64
	Status           BookingStatus
	BasePrice        float64
	TotalPrice       float64
	Currency         string
	CreatedAt        time.Time
}

type Passenger struct {
	ID             int64
	BookingID      int64
	Title          string
	FirstName      string
	LastName       string
	DateOfBirth    string
	Nationality    string
	PassportNumber string
	PassportExpiry string
	TicketNumber   string
	CreatedAt      time.Time
}

type Quote struct {
	FlightInstanceID int64
	FareClassCode    string
	Passengers       int
	TotalAmount      float64
	ExpiresAt        time.Time
}
