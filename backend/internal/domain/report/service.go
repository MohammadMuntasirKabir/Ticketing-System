package report

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/domain/booking"
	"github.com/you/airline-ticketing/internal/domain/flight"
	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

// FlightProvider is the slice of the flight repository the reports
// service needs.
type FlightProvider interface {
	ListFlightInstances(ctx context.Context, query flight.ListQuery) ([]flight.FlightInstance, error)
	AircraftSeatCount(ctx context.Context, flightInstanceID int64) (int, error)
	FlightInfo(ctx context.Context, flightInstanceID int64) (flightNumber, origin, destination string, err error)
}

// BookingProvider is the slice of the booking repository the reports
// service needs.
type BookingProvider interface {
	ListBookings(ctx context.Context, limit, offset int) ([]booking.Booking, error)
}

type Service struct {
	flights  FlightProvider
	bookings BookingProvider
	log      *zap.Logger
}

func NewService(flights FlightProvider, bookings BookingProvider, log *zap.Logger) *Service {
	return &Service{flights: flights, bookings: bookings, log: log}
}

type LoadFactorRow struct {
	FlightInstanceID int64   `json:"flight_instance_id"`
	FlightNumber     string  `json:"flight_number"`
	Departure        string  `json:"departure"`
	Capacity         int     `json:"capacity"`
	Sold             int     `json:"sold"`
	LoadFactor       float64 `json:"load_factor"`
}

type RevenueRow struct {
	FlightInstanceID int64   `json:"flight_instance_id"`
	FlightNumber     string  `json:"flight_number"`
	Departure        string  `json:"departure"`
	Bookings         int     `json:"bookings"`
	Revenue          float64 `json:"revenue"`
}

type PeakTimesRow struct {
	Hour     int `json:"hour"`
	Bookings int `json:"bookings"`
}

// LoadFactor reports sold seats vs capacity per flight instance.
func (s *Service) LoadFactor(ctx context.Context) ([]LoadFactorRow, error) {
	instances, err := s.flights.ListFlightInstances(ctx, flight.ListQuery{Limit: 100})
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list flights", err)
	}
	bookings, err := s.bookings.ListBookings(ctx, 0, 0)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list bookings", err)
	}
	soldByFlight := map[int64]int{}
	for _, b := range bookings {
		if b.Status == booking.BookingConfirmed {
			soldByFlight[b.FlightInstanceID]++
		}
	}

	var rows []LoadFactorRow
	for _, inst := range instances {
		capacity, err := s.flights.AircraftSeatCount(ctx, inst.ID)
		if err != nil {
			continue
		}
		sold := soldByFlight[inst.ID]
		load := 0.0
		if capacity > 0 {
			load = float64(sold) / float64(capacity) * 100
		}
		fn, _, _, _ := s.flights.FlightInfo(ctx, inst.ID)
		rows = append(rows, LoadFactorRow{
			FlightInstanceID: inst.ID,
			FlightNumber:     fn,
			Departure:        inst.DepartureTime.Format(time.RFC3339),
			Capacity:         capacity,
			Sold:             sold,
			LoadFactor:       round2(load),
		})
	}
	return rows, nil
}

// Revenue reports confirmed booking value per flight instance.
func (s *Service) Revenue(ctx context.Context) ([]RevenueRow, error) {
	instances, err := s.flights.ListFlightInstances(ctx, flight.ListQuery{Limit: 100})
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list flights", err)
	}
	bookings, err := s.bookings.ListBookings(ctx, 0, 0)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list bookings", err)
	}

	revenueByFlight := map[int64]*RevenueRow{}
	for _, b := range bookings {
		if b.Status != booking.BookingConfirmed {
			continue
		}
		row := revenueByFlight[b.FlightInstanceID]
		if row == nil {
			row = &RevenueRow{FlightInstanceID: b.FlightInstanceID}
			revenueByFlight[b.FlightInstanceID] = row
		}
		row.Bookings++
		row.Revenue += b.TotalPrice
	}

	var rows []RevenueRow
	for _, inst := range instances {
		row := revenueByFlight[inst.ID]
		if row == nil {
			continue
		}
		fn, _, _, _ := s.flights.FlightInfo(ctx, inst.ID)
		row.FlightNumber = fn
		row.Departure = inst.DepartureTime.Format(time.RFC3339)
		row.Revenue = round2(row.Revenue)
		rows = append(rows, *row)
	}
	return rows, nil
}

// PeakTimes buckets confirmed bookings by their flight's departure hour.
func (s *Service) PeakTimes(ctx context.Context) ([]PeakTimesRow, error) {
	instances, err := s.flights.ListFlightInstances(ctx, flight.ListQuery{Limit: 100})
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list flights", err)
	}
	bookings, err := s.bookings.ListBookings(ctx, 0, 0)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list bookings", err)
	}

	departureHour := map[int64]int{}
	for _, inst := range instances {
		departureHour[inst.ID] = inst.DepartureTime.Hour()
	}

	buckets := map[int]int{}
	for _, b := range bookings {
		if b.Status != booking.BookingConfirmed {
			continue
		}
		hour, ok := departureHour[b.FlightInstanceID]
		if !ok {
			continue
		}
		buckets[hour]++
	}

	var rows []PeakTimesRow
	for hour, n := range buckets {
		rows = append(rows, PeakTimesRow{Hour: hour, Bookings: n})
	}
	for i := 1; i < len(rows); i++ {
		for j := 0; j < len(rows)-i; j++ {
			if rows[j].Hour > rows[j+1].Hour {
				rows[j], rows[j+1] = rows[j+1], rows[j]
			}
		}
	}
	return rows, nil
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
