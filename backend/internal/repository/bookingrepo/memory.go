package bookingrepo

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/you/airline-ticketing/internal/domain/booking"
)

var errNotFound = errors.New("bookingrepo: not found")

// Memory is an in-memory booking.Repository used for local development
// and tests.
type Memory struct {
	mu            sync.RWMutex
	pnrs          map[int64]booking.PNR
	pnrsByCode    map[string]int64
	bookings      map[int64]booking.Booking
	bookingsByPNR map[int64][]int64
	passengers    map[int64][]booking.Passenger
	nextPNR       int64
	nextBooking   int64
	nextPassenger int64
}

func New() *Memory {
	return &Memory{
		pnrs:          map[int64]booking.PNR{},
		pnrsByCode:    map[string]int64{},
		bookings:      map[int64]booking.Booking{},
		bookingsByPNR: map[int64][]int64{},
		passengers:    map[int64][]booking.Passenger{},
		nextPNR:       1,
		nextBooking:   1,
		nextPassenger: 1,
	}
}

func (m *Memory) CreatePNR(ctx context.Context, p *booking.PNR) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.pnrsByCode[p.Code]; exists {
		return 0, errors.New("pnr code already exists")
	}
	id := m.nextPNR
	m.nextPNR++
	p.ID = id
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = time.Now().UTC()
	m.pnrs[id] = *p
	m.pnrsByCode[p.Code] = id
	return id, nil
}

func (m *Memory) GetPNRByCode(ctx context.Context, code string) (*booking.PNR, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.pnrsByCode[code]
	if !ok {
		return nil, nil
	}
	p, ok := m.pnrs[id]
	if !ok {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

func (m *Memory) GetPNR(ctx context.Context, id int64) (*booking.PNR, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.pnrs[id]
	if !ok {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

func (m *Memory) UpdatePNRStatus(ctx context.Context, pnrID int64, status booking.PNRStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pnrs[pnrID]
	if !ok {
		return errNotFound
	}
	p.Status = status
	p.UpdatedAt = time.Now().UTC()
	m.pnrs[pnrID] = p
	return nil
}

func (m *Memory) DeletePNR(ctx context.Context, pnrID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pnrs[pnrID]
	if !ok {
		return errNotFound
	}
	delete(m.pnrs, pnrID)
	delete(m.pnrsByCode, p.Code)
	for _, bid := range m.bookingsByPNR[pnrID] {
		delete(m.bookings, bid)
		delete(m.passengers, bid)
	}
	delete(m.bookingsByPNR, pnrID)
	return nil
}

func (m *Memory) CreateBooking(ctx context.Context, b *booking.Booking) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextBooking
	m.nextBooking++
	b.ID = id
	b.CreatedAt = time.Now().UTC()
	m.bookings[id] = *b
	m.bookingsByPNR[b.PNRID] = append(m.bookingsByPNR[b.PNRID], id)
	return id, nil
}

func (m *Memory) GetBookingsByPNR(ctx context.Context, pnrID int64) ([]booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := m.bookingsByPNR[pnrID]
	var out []booking.Booking
	for _, id := range ids {
		out = append(out, m.bookings[id])
	}
	return out, nil
}

func (m *Memory) GetBooking(ctx context.Context, id int64) (*booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.bookings[id]
	if !ok {
		return nil, nil
	}
	cp := b
	return &cp, nil
}

func (m *Memory) UpdateBookingStatus(ctx context.Context, id int64, status booking.BookingStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bookings[id]
	if !ok {
		return errNotFound
	}
	b.Status = status
	m.bookings[id] = b
	return nil
}

func (m *Memory) UpdateBookingSeat(ctx context.Context, bookingID int64, seatID *int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bookings[bookingID]
	if !ok {
		return errNotFound
	}
	b.SeatID = seatID
	m.bookings[bookingID] = b
	return nil
}

func (m *Memory) GetPassengersByBooking(ctx context.Context, bookingID int64) ([]booking.Passenger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := append([]booking.Passenger(nil), m.passengers[bookingID]...)
	return out, nil
}

func (m *Memory) ListBookings(ctx context.Context, limit, offset int) ([]booking.Booking, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []booking.Booking
	for _, b := range m.bookings {
		out = append(out, b)
	}
	if offset >= len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// AddPassengers is a convenience used by tests to attach passenger
// records to a booking.
func (m *Memory) AddPassengers(bookingID int64, passengers ...booking.Passenger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range passengers {
		passengers[i].ID = m.nextPassenger
		m.nextPassenger++
		m.passengers[bookingID] = append(m.passengers[bookingID], passengers[i])
	}
}
