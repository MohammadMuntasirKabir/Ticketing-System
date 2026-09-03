package checkin

import "time"

// CheckIn records that a passenger has checked in for a PNR.
type CheckIn struct {
	ID            int64     `json:"id"`
	PNRID         int64     `json:"pnr_id"`
	BookingID     int64     `json:"booking_id"`
	CheckedInAt   time.Time `json:"checked_in_at"`
	IsOnline      bool      `json:"is_online"`
	BaggageCount  int       `json:"baggage_count"`
	BaggageWeight float64   `json:"baggage_weight"`
}

// BoardingPass is the printable/QR document issued after seat assignment.
type BoardingPass struct {
	ID             int64     `json:"id"`
	CheckInID      int64     `json:"check_in_id"`
	PassengerID    int64     `json:"passenger_id"`
	FlightNumber   string    `json:"flight_number"`
	Origin         string    `json:"origin"`
	Destination    string    `json:"destination"`
	SeatNumber     string    `json:"seat_number"`
	BoardingGroup  string    `json:"boarding_group"`
	SequenceNumber int       `json:"sequence_number"`
	Gate           string    `json:"gate"`
	BoardingTime   time.Time `json:"boarding_time"`
	QRCodeData     string    `json:"qr_code_data"`
	IssuedAt       time.Time `json:"issued_at"`
}
