package flight

import (
	"context"
	"time"
)

// Repository abstracts persistence for flights, airports, airlines,
// aircraft and routes. Concrete implementations live in the repository
// layer (postgres for production, in-memory for tests).
type Repository interface {
	Search(ctx context.Context, criteria SearchCriteria) ([]SearchResult, error)
	GetFlightInstance(ctx context.Context, id int64) (*FlightInstance, error)
	GetFlightSchedule(ctx context.Context, id int64) (*FlightSchedule, error)
	GetSeatMap(ctx context.Context, flightInstanceID int64) (*SeatMapData, error)

	ListAirports(ctx context.Context, query string) ([]Airport, error)
	ListAirlines(ctx context.Context) ([]Airline, error)
	ListAircraft(ctx context.Context) ([]Aircraft, error)
	ListRoutes(ctx context.Context) ([]Route, error)
	ListFareClasses(ctx context.Context, flightInstanceID int64) ([]FareAvailability, error)

	CreateRoute(ctx context.Context, r RouteInput) (int64, error)
	CreateFlightSchedule(ctx context.Context, s ScheduleInput) (int64, error)
	MaterializeSchedule(ctx context.Context, scheduleID int64, date time.Time) (int, error)
	UpdateFlightInstance(ctx context.Context, id int64, f FlightInstanceUpdate) error
	DeleteFlightInstance(ctx context.Context, id int64) error
	ListFlightInstances(ctx context.Context, query ListQuery) ([]FlightInstance, error)

	// ResolveFareClass maps a fare-class code on a flight instance to its
	// stable ID and seat capacity. Used by the booking orchestrator.
	ResolveFareClass(ctx context.Context, flightInstanceID int64, code string) (fareClassID int64, capacity int, err error)

	// FareClassCapacity returns the total seat capacity of a fare class on
	// a flight instance. Used by the inventory service.
	FareClassCapacity(ctx context.Context, flightInstanceID, fareClassID int64) (int, error)

	// AircraftSeatCount returns the total number of seats on the aircraft
	// operating a flight instance. Used by the reports service.
	AircraftSeatCount(ctx context.Context, flightInstanceID int64) (int, error)

	// FlightInfo returns the flight number plus origin/destination airport
	// codes for boarding passes. Used by check-in.
	FlightInfo(ctx context.Context, flightInstanceID int64) (flightNumber, origin, destination string, err error)
}

// FareAvailability couples a fare class with its seat capacity so the
// search listing can report how many seats remain.
type FareAvailability struct {
	FareClass
	Available int
}

type RouteInput struct {
	AirlineID     int64 `validate:"required"`
	OriginID      int64 `validate:"required"`
	DestinationID int64 `validate:"required"`
	DistanceKM    *float64
	BasePrice     float64 `validate:"gte=0"`
	Currency      string  `validate:"required,len=3"`
}

type ScheduleInput struct {
	RouteID       int64     `validate:"required"`
	AircraftID    int64     `validate:"required"`
	FlightNumber  string    `validate:"required"`
	DepartureTime time.Time `validate:"required"`
	ArrivalTime   time.Time `validate:"required"`
	DaysOfWeek    int
}

type FlightInstanceUpdate struct {
	DepartureTime *time.Time
	ArrivalTime   *time.Time
	DelayMinutes  *int
	Status        *string
}

type ListQuery struct {
	Limit  int
	Offset int
}
