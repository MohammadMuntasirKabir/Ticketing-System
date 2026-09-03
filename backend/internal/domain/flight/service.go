package flight

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
)

// Service is the flight use-case. It is intentionally thin: it enforces
// domain rules (e.g. search validation) and delegates persistence to the
// repository, keeping it trivially unit-testable with a fake repo.
type Service struct {
	log  *zap.Logger
	repo Repository
}

func NewService(repo Repository, log *zap.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) Search(ctx context.Context, criteria SearchCriteria) ([]SearchResult, error) {
	if strings.TrimSpace(string(criteria.Origin)) == "" || strings.TrimSpace(string(criteria.Destination)) == "" {
		return nil, domainErrors.Validation("origin and destination are required")
	}
	if criteria.Origin == criteria.Destination {
		return nil, domainErrors.Validation("origin and destination must differ")
	}
	if criteria.Passengers < 1 {
		return nil, domainErrors.Validation("at least one passenger is required")
	}
	if !criteria.Date.IsZero() {
		// Reject past travel dates (same-day departures are allowed).
		today := time.Now().Truncate(24 * time.Hour)
		if criteria.Date.Before(today) {
			return nil, domainErrors.Validation("travel date cannot be in the past")
		}
	}

	results, err := s.repo.Search(ctx, criteria)
	if err != nil {
		return nil, domainErrors.InternalWrap("flight search failed", err)
	}
	return results, nil
}

func (s *Service) GetDetail(ctx context.Context, id int64) (*FlightInstance, error) {
	if id <= 0 {
		return nil, domainErrors.Validation("flight instance id is required")
	}
	fi, err := s.repo.GetFlightInstance(ctx, id)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to load flight instance", err)
	}
	if fi == nil {
		return nil, domainErrors.NotFound("flight instance not found")
	}
	return fi, nil
}

func (s *Service) GetSeatMap(ctx context.Context, id int64) (*SeatMapData, error) {
	if id <= 0 {
		return nil, domainErrors.Validation("flight instance id is required")
	}
	sm, err := s.repo.GetSeatMap(ctx, id)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to build seat map", err)
	}
	return sm, nil
}

func (s *Service) ListFlightInstances(ctx context.Context, query ListQuery) ([]FlightInstance, error) {
	if query.Limit <= 0 {
		query.Limit = 100
	}
	instances, err := s.repo.ListFlightInstances(ctx, query)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list flight instances", err)
	}
	return instances, nil
}

func (s *Service) ListAirports(ctx context.Context, query string) ([]Airport, error) {
	airports, err := s.repo.ListAirports(ctx, query)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list airports", err)
	}
	return airports, nil
}

func (s *Service) ListAirlines(ctx context.Context) ([]Airline, error) {
	airlines, err := s.repo.ListAirlines(ctx)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list airlines", err)
	}
	return airlines, nil
}

func (s *Service) ListAircraft(ctx context.Context) ([]Aircraft, error) {
	ac, err := s.repo.ListAircraft(ctx)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list aircraft", err)
	}
	return ac, nil
}

func (s *Service) ListRoutes(ctx context.Context) ([]Route, error) {
	rs, err := s.repo.ListRoutes(ctx)
	if err != nil {
		return nil, domainErrors.InternalWrap("failed to list routes", err)
	}
	return rs, nil
}

func (s *Service) CreateRoute(ctx context.Context, in RouteInput) (int64, error) {
	id, err := s.repo.CreateRoute(ctx, in)
	if err != nil {
		return 0, domainErrors.InternalWrap("failed to create route", err)
	}
	return id, nil
}

func (s *Service) CreateFlightSchedule(ctx context.Context, in ScheduleInput) (int64, error) {
	if in.DepartureTime.After(in.ArrivalTime) || in.DepartureTime.Equal(in.ArrivalTime) {
		return 0, domainErrors.Validation("departure must be before arrival")
	}
	id, err := s.repo.CreateFlightSchedule(ctx, in)
	if err != nil {
		return 0, domainErrors.InternalWrap("failed to create flight schedule", err)
	}
	return id, nil
}

func (s *Service) MaterializeSchedule(ctx context.Context, scheduleID int64, date time.Time) (int, error) {
	if scheduleID <= 0 {
		return 0, domainErrors.Validation("schedule id is required")
	}
	if date.IsZero() {
		return 0, domainErrors.Validation("date is required")
	}
	n, err := s.repo.MaterializeSchedule(ctx, scheduleID, date)
	if err != nil {
		return 0, domainErrors.InternalWrap("materialization failed", err)
	}
	return n, nil
}

func (s *Service) UpdateFlightInstance(ctx context.Context, id int64, in FlightInstanceUpdate) error {
	if id <= 0 {
		return domainErrors.Validation("flight instance id is required")
	}
	return s.repo.UpdateFlightInstance(ctx, id, in)
}

func (s *Service) DeleteFlightInstance(ctx context.Context, id int64) error {
	if id <= 0 {
		return domainErrors.Validation("flight instance id is required")
	}
	if err := s.repo.DeleteFlightInstance(ctx, id); err != nil {
		return domainErrors.InternalWrap("failed to delete flight instance", err)
	}
	return nil
}
