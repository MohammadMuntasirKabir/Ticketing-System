package inventory

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	domainErrors "github.com/you/airline-ticketing/internal/pkg/errors"
	"github.com/you/airline-ticketing/internal/pkg/store"
)

// CapacityProvider returns the total seat capacity of a fare class for a
// flight instance. The flight repository satisfies this interface, which
// keeps the inventory service decoupled from the flight domain.
type CapacityProvider interface {
	FareClassCapacity(ctx context.Context, flightInstanceID, fareClassID int64) (int, error)
}

// Service tracks real-time seat availability using a capacity counter
// (sourced from CapacityProvider) and per-PNR reservation records stored
// in a KV store. Reservations and confirmed bookings both reduce
// availability until explicitly released (on cancellation).
type Service struct {
	kv      store.KV
	cap     CapacityProvider
	log     *zap.Logger
	lockTTL time.Duration
	holdTTL time.Duration
}

func NewService(kv store.KV, capProv CapacityProvider, log *zap.Logger) *Service {
	return &Service{
		kv:      kv,
		cap:     capProv,
		log:     log,
		lockTTL: 10 * time.Second,
		holdTTL: 15 * time.Minute,
	}
}

type HoldRequest struct {
	FlightInstanceID int64
	FareClassID      int64
	Passengers       int
	UserID           int64
	PNRCode          string
}

func lockKey(flightInstanceID, fareClassID int64) string {
	return fmt.Sprintf("inv:lock:%d:%d", flightInstanceID, fareClassID)
}

func holdKey(flightInstanceID, fareClassID int64, pnrCode string) string {
	return fmt.Sprintf("inv:hold:%d:%d:%s", flightInstanceID, fareClassID, pnrCode)
}

// Available returns how many seats remain for the fare class. Confirmed
// bookings keep consuming capacity until the reservation is released.
func (s *Service) Available(ctx context.Context, flightInstanceID, fareClassID int64) (int, error) {
	capacity, err := s.cap.FareClassCapacity(ctx, flightInstanceID, fareClassID)
	if err != nil {
		return 0, domainErrors.InternalWrap("failed to read capacity", err)
	}
	reserved, err := s.reservedCount(ctx, flightInstanceID, fareClassID)
	if err != nil {
		return 0, err
	}
	avail := capacity - reserved
	if avail < 0 {
		return 0, nil
	}
	return avail, nil
}

func (s *Service) reservedCount(ctx context.Context, flightInstanceID, fareClassID int64) (int, error) {
	pattern := fmt.Sprintf("inv:hold:%d:%d:*", flightInstanceID, fareClassID)
	keys, err := s.kv.Keys(ctx, pattern)
	if err != nil {
		return 0, domainErrors.InternalWrap("failed to enumerate holds", err)
	}
	var total int
	for _, k := range keys {
		v, err := s.kv.GetString(ctx, k)
		if err != nil {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		total += n
	}
	return total, nil
}

// HoldSeats atomically reserves `passengers` seats. It returns an
// inventory error on contention (another in-flight hold) or when not
// enough seats remain.
func (s *Service) HoldSeats(ctx context.Context, req HoldRequest) error {
	if req.Passengers < 1 {
		return domainErrors.Validation("at least one passenger is required")
	}
	lk := lockKey(req.FlightInstanceID, req.FareClassID)
	held, err := s.kv.SetNX(ctx, lk, "1", s.lockTTL)
	if err != nil {
		s.log.Error("inventory lock error", zap.Error(err))
		return domainErrors.InternalWrap("failed to acquire inventory lock", err)
	}
	if !held {
		return domainErrors.ErrInventoryContention
	}
	defer func() { _ = s.kv.Del(ctx, lk) }()

	avail, err := s.Available(ctx, req.FlightInstanceID, req.FareClassID)
	if err != nil {
		return err
	}
	if avail < req.Passengers {
		return domainErrors.ErrInventoryConflict
	}

	stored, err := s.kv.SetNX(ctx, holdKey(req.FlightInstanceID, req.FareClassID, req.PNRCode), strconv.Itoa(req.Passengers), s.holdTTL)
	if err != nil {
		return domainErrors.InternalWrap("failed to record seat hold", err)
	}
	if !stored {
		return domainErrors.ErrInventoryConflict
	}
	s.log.Debug("seats held",
		zap.Int64("flight_instance_id", req.FlightInstanceID),
		zap.Int64("fare_class_id", req.FareClassID),
		zap.String("pnr_code", req.PNRCode),
		zap.Int("passengers", req.Passengers))
	return nil
}

// ReleaseHolds returns held seats to the available pool for a PNR.
func (s *Service) ReleaseHolds(ctx context.Context, pnrCode string) error {
	return s.adjustHolds(ctx, pnrCode)
}

// ConfirmHolds marks a PNR's reservations as confirmed by extending the
// hold TTL indefinitely: a confirmed booking keeps consuming capacity
// until the booking is cancelled (which deletes the hold key).
func (s *Service) ConfirmHolds(ctx context.Context, pnrCode string) error {
	pattern := fmt.Sprintf("inv:hold:*:*:%s", pnrCode)
	keys, err := s.kv.Keys(ctx, pattern)
	if err != nil {
		return domainErrors.InternalWrap("failed to confirm holds", err)
	}
	for _, k := range keys {
		if v, gerr := s.kv.GetString(ctx, k); gerr == nil {
			_ = s.kv.Set(ctx, k, v, 0)
		}
	}
	return nil
}

func (s *Service) adjustHolds(ctx context.Context, pnrCode string) error {
	pattern := fmt.Sprintf("inv:hold:*:*:%s", pnrCode)
	keys, err := s.kv.Keys(ctx, pattern)
	if err != nil {
		return domainErrors.InternalWrap("failed to find holds", err)
	}
	for _, k := range keys {
		_ = s.kv.Del(ctx, k)
	}
	return nil
}
