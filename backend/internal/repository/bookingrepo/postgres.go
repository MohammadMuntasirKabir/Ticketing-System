package bookingrepo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/you/airline-ticketing/internal/domain/booking"
)

// Postgres implements booking.Repository using PostgreSQL.
type Postgres struct {
	db *sql.DB
}

// NewPostgres creates a new PostgreSQL-backed booking repository.
func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (r *Postgres) CreatePNR(ctx context.Context, p *booking.PNR) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO pnrs (code, user_id, status, total_amount, currency, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) RETURNING id
	`, p.Code, p.UserID, string(p.Status), p.TotalAmount, p.Currency, p.ExpiresAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create PNR: %w", err)
	}
	return id, nil
}

func (r *Postgres) GetPNRByCode(ctx context.Context, code string) (*booking.PNR, error) {
	var p booking.PNR
	var status string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, user_id, status, total_amount, currency, expires_at, created_at, updated_at
		WHERE code = $1
	`, code).Scan(&p.ID, &p.Code, &p.UserID, &status, &p.TotalAmount, &p.Currency, &p.ExpiresAt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get PNR by code: %w", err)
	}
	p.Status = booking.PNRStatus(status)
	return &p, nil
}

func (r *Postgres) GetPNR(ctx context.Context, id int64) (*booking.PNR, error) {
	var p booking.PNR
	var status string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, user_id, status, total_amount, currency, expires_at, created_at, updated_at
		WHERE id = $1
	`, id).Scan(&p.ID, &p.Code, &p.UserID, &status, &p.TotalAmount, &p.Currency, &p.ExpiresAt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get PNR: %w", err)
	}
	p.Status = booking.PNRStatus(status)
	return &p, nil
}

func (r *Postgres) UpdatePNRStatus(ctx context.Context, pnrID int64, status booking.PNRStatus) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE pnrs SET status = $1, updated_at = NOW() WHERE id = $2
	`, string(status), pnrID)
	return err
}

func (r *Postgres) DeletePNR(ctx context.Context, pnrID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM pnrs WHERE id = $1`, pnrID)
	return err
}

func (r *Postgres) CreateBooking(ctx context.Context, b *booking.Booking) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO bookings (pnr_id, flight_instance_id, fare_class_id, seat_id, status, base_price, total_price, currency, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW()) RETURNING id
	`, b.PNRID, b.FlightInstanceID, b.FareClassID, b.SeatID, string(b.Status), b.BasePrice, b.TotalPrice, b.Currency).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create booking: %w", err)
	}
	return id, nil
}

func (r *Postgres) GetBookingsByPNR(ctx context.Context, pnrID int64) ([]booking.Booking, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, pnr_id, flight_instance_id, fare_class_id, seat_id, status, base_price, total_price, currency, created_at
		WHERE pnr_id = $1
	`, pnrID)
	if err != nil {
		return nil, fmt.Errorf("get bookings by PNR: %w", err)
	}
	defer rows.Close()
	return scanBookings(rows)
}

func (r *Postgres) GetBooking(ctx context.Context, id int64) (*booking.Booking, error) {
	var b booking.Booking
	var status string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, pnr_id, flight_instance_id, fare_class_id, seat_id, status, base_price, total_price, currency, created_at
		WHERE id = $1
	`, id).Scan(&b.ID, &b.PNRID, &b.FlightInstanceID, &b.FareClassID, &b.SeatID, &status, &b.BasePrice, &b.TotalPrice, &b.Currency, &b.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get booking: %w", err)
	}
	b.Status = booking.BookingStatus(status)
	return &b, nil
}

func (r *Postgres) UpdateBookingStatus(ctx context.Context, id int64, status booking.BookingStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE bookings SET status = $1 WHERE id = $2`, string(status), id)
	return err
}

func (r *Postgres) UpdateBookingSeat(ctx context.Context, bookingID int64, seatID *int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE bookings SET seat_id = $1 WHERE id = $2`, seatID, bookingID)
	return err
}

func (r *Postgres) GetPassengersByBooking(ctx context.Context, bookingID int64) ([]booking.Passenger, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, booking_id, title, first_name, last_name, date_of_birth, nationality, passport_number, passport_expiry, ticket_number, created_at
		WHERE booking_id = $1
	`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("get passengers: %w", err)
	}
	defer rows.Close()
	var results []booking.Passenger
	for rows.Next() {
		var p booking.Passenger
		if err := rows.Scan(&p.ID, &p.BookingID, &p.Title, &p.FirstName, &p.LastName, &p.DateOfBirth, &p.Nationality, &p.PassportNumber, &p.PassportExpiry, &p.TicketNumber, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan passenger: %w", err)
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

func (r *Postgres) ListBookings(ctx context.Context, limit, offset int) ([]booking.Booking, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, pnr_id, flight_instance_id, fare_class_id, seat_id, status, base_price, total_price, currency, created_at
		ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()
	return scanBookings(rows)
}

func scanBookings(rows *sql.Rows) ([]booking.Booking, error) {
	var results []booking.Booking
	for rows.Next() {
		var b booking.Booking
		var status string
		if err := rows.Scan(&b.ID, &b.PNRID, &b.FlightInstanceID, &b.FareClassID, &b.SeatID, &status, &b.BasePrice, &b.TotalPrice, &b.Currency, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		b.Status = booking.BookingStatus(status)
		results = append(results, b)
	}
	return results, rows.Err()
}

var _ booking.Repository = (*Postgres)(nil)
