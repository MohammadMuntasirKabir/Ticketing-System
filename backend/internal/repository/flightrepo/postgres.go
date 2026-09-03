package flightrepo

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/you/airline-ticketing/internal/domain/flight"
)

// Postgres implements flight.Repository using PostgreSQL.
type Postgres struct {
	db *sql.DB
}

// NewPostgres creates a new PostgreSQL-backed flight repository.
func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}

func (r *Postgres) Search(ctx context.Context, criteria flight.SearchCriteria) ([]flight.SearchResult, error) {
	query := `
		SELECT fi.id, fi.schedule_id, fi.departure_date, fi.departure_time, fi.arrival_time,
		       fi.status, fi.delay_minutes, fi.created_at,
		       fs.flight_number, fs.route_id, fs.aircraft_id,
		       r.airline_id, r.base_price, r.currency, r.distance_km,
		       al.code, al.name, al.country, al.status,
		       orig.code, orig.name, orig.city, orig.country, orig.timezone, orig.status,
		       dest.code, dest.name, dest.city, dest.country, dest.timezone, dest.status,
		       ac.model, ac.registration, ac.total_seats, ac.airline_id, ac.status
		FROM flight_instances fi
		JOIN flight_schedules fs ON fi.schedule_id = fs.id
		JOIN routes r ON fs.route_id = r.id
		JOIN airlines al ON r.airline_id = al.id
		JOIN airports orig ON r.origin_id = orig.id
		JOIN airports dest ON r.destination_id = dest.id
		JOIN aircraft ac ON fs.aircraft_id = ac.id
		WHERE orig.code = $1 AND dest.code = $2 AND fi.status = 'SCHEDULED'
	`
	args := []interface{}{string(criteria.Origin), string(criteria.Destination)}
	if !criteria.Date.IsZero() {
		query += " AND fi.departure_date = $3"
		args = append(args, criteria.Date)
	}
	if criteria.CabinClass != "" {
		query += " AND EXISTS (SELECT 1 FROM flight_instance_fare_classes fic JOIN fare_classes fc ON fic.fare_class_id = fc.id JOIN cabin_classes cc ON fc.cabin_class_id = cc.id WHERE fic.flight_instance_id = fi.id AND cc.code = $4)"
		args = append(args, criteria.CabinClass)
	}
	if criteria.Passengers > 0 {
		query += " AND fi.available_seats >= $5"
		args = append(args, criteria.Passengers)
	}
	query += " ORDER BY fi.departure_time"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search flights: %w", err)
	}
	defer rows.Close()

	var results []flight.SearchResult
	for rows.Next() {
		var fi flight.FlightInstance
		var schedID, aircraftID int64
		var flightNum string
		var airlineID int64
		var basePrice float64
		var currency string
		var distKM *float64
		var alCode flight.AirlineCode
		var alName, alCountry, alStatus string
		var origCode flight.AirportCode
		var origName, origCity, origCountry, origTZ, origStatus string
		var destCode flight.AirportCode
		var destName, destCity, destCountry, destTZ, destStatus string
		var acModel, acReg string
		var acSeats int
		var acAirlineID int64
		var acStatus string

		err := rows.Scan(
			&fi.ID, &schedID, &fi.DepartureDate, &fi.DepartureTime, &fi.ArrivalTime,
			&fi.Status, &fi.DelayMinutes, &fi.CreatedAt,
			&flightNum, &fi.ScheduleID, &aircraftID,
			&airlineID, &basePrice, &currency, &distKM,
			&alCode, &alName, &alCountry, &alStatus,
			&origCode, &origName, &origCity, &origCountry, &origTZ, &origStatus,
			&destCode, &destName, &destCity, &destCountry, &destTZ, &destStatus,
			&acModel, &acReg, &acSeats, &acAirlineID, &acStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("scan flight: %w", err)
		}

		airline := flight.Airline{ID: airlineID, Code: alCode, Name: alName, Country: alCountry, Status: alStatus}
		origin := flight.Airport{ID: 0, Code: origCode, Name: origName, City: origCity, Country: origCountry, Timezone: origTZ, Status: origStatus}
		dest := flight.Airport{ID: 0, Code: destCode, Name: destName, City: destCity, Country: destCountry, Timezone: destTZ, Status: destStatus}

		fares, err := r.ListFareClasses(ctx, fi.ID)
		if err != nil {
			return nil, fmt.Errorf("get fares: %w", err)
		}

		var fareOptions []flight.FareOption
		for _, fa := range fares {
			cabinCode := string(r.cabinCode(ctx, fa.CabinID))
			if criteria.CabinClass != "" && !strings.EqualFold(cabinCode, criteria.CabinClass) {
				continue
			}
			base := r.fareClassBasePrice(fa.Code)
			taxes := base*0.075 + 4.50 + 5.60
			fees := base * 0.05
			fareOptions = append(fareOptions, flight.FareOption{
				FareClassCode: fa.Code,
				Cabin:         cabinCode,
				Available:     fa.Available,
				Price:         fareRound2(base),
				Taxes:         fareRound2(taxes),
				Total:         fareRound2(base + taxes + fees),
			})
		}

		if len(fareOptions) == 0 {
			continue
		}

		results = append(results, flight.SearchResult{
			FlightID:     flight.ID(fi.ID),
			Airline:      airline,
			FlightNumber: flightNum,
			Origin:       origin,
			Destination:  dest,
			Departure:    fi.DepartureTime,
			Arrival:      fi.ArrivalTime,
			DurationMins: int(fi.ArrivalTime.Sub(fi.DepartureTime).Minutes()),
			Aircraft:     acModel,
			Fares:        fareOptions,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Departure.Before(results[j].Departure)
	})
	return results, rows.Err()
}

func (r *Postgres) GetFlightInstance(ctx context.Context, id int64) (*flight.FlightInstance, error) {
	var fi flight.FlightInstance
	err := r.db.QueryRowContext(ctx, `
		SELECT id, schedule_id, departure_date, departure_time, arrival_time, status, delay_minutes, created_at
		FROM flight_instances WHERE id = $1
	`, id).Scan(&fi.ID, &fi.ScheduleID, &fi.DepartureDate, &fi.DepartureTime, &fi.ArrivalTime, &fi.Status, &fi.DelayMinutes, &fi.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get flight instance: %w", err)
	}
	return &fi, nil
}

func (r *Postgres) GetFlightSchedule(ctx context.Context, id int64) (*flight.FlightSchedule, error) {
	var fs flight.FlightSchedule
	err := r.db.QueryRowContext(ctx, `
		SELECT id, route_id, aircraft_id, flight_number, departure_time, arrival_time, days_of_week, status, created_at
		FROM flight_schedules WHERE id = $1
	`, id).Scan(&fs.ID, &fs.RouteID, &fs.AircraftID, &fs.FlightNumber, &fs.DepartureTime, &fs.ArrivalTime, &fs.DaysOfWeek, &fs.Status, &fs.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get flight schedule: %w", err)
	}
	return &fs, nil
}

func (r *Postgres) GetSeatMap(ctx context.Context, flightInstanceID int64) (*flight.SeatMapData, error) {
	fi, err := r.GetFlightInstance(ctx, flightInstanceID)
	if err != nil {
		return nil, err
	}
	if fi == nil {
		return nil, fmt.Errorf("flight instance not found: %d", flightInstanceID)
	}

	sched, err := r.GetFlightSchedule(ctx, fi.ScheduleID)
	if err != nil {
		return nil, err
	}

	var acModel, acReg string
	err = r.db.QueryRowContext(ctx, `SELECT model, registration FROM aircraft WHERE id = $1`, sched.AircraftID).Scan(&acModel, &acReg)
	if err != nil {
		return nil, fmt.Errorf("get aircraft: %w", err)
	}

	seatRows, err := r.db.QueryContext(ctx, `
		SELECT s.id, s.row_num, s.col, s.position, s.cabin_class_id, s.is_emergency_exit, s.is_extra_legroom, cc.code
		FROM seats s JOIN cabin_classes cc ON s.cabin_class_id = cc.id
		WHERE s.aircraft_id = $1 AND s.is_active = true
		ORDER BY s.row_num, s.column_letter
	`, sched.AircraftID)
	if err != nil {
		return nil, fmt.Errorf("get seats: %w", err)
	}
	defer seatRows.Close()

	var seats []flight.SeatInfo
	for seatRows.Next() {
		var s flight.SeatInfo
		var cabinCode string
		if err := seatRows.Scan(&s.ID, &s.Row, &s.Col, &s.Position, &s.Cabin, &s.IsEmergencyExit, &s.IsExtraLegroom, &cabinCode); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		s.Cabin = cabinCode
		s.IsAvailable = true
		seats = append(seats, s)
	}

	sectionRows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT cc.code, cc.name, MIN(s.row_num), MAX(s.row_num), ARRAY_AGG(DISTINCT s.column_letter ORDER BY s.column_letter)
		FROM seats s JOIN cabin_classes cc ON s.cabin_class_id = cc.id
		WHERE s.aircraft_id = $1 AND s.is_active = true
		GROUP BY cc.code, cc.name ORDER BY MIN(s.row_num)
	`, sched.AircraftID)
	if err != nil {
		return nil, fmt.Errorf("get sections: %w", err)
	}
	defer sectionRows.Close()

	var sections []flight.CabinSection
	for sectionRows.Next() {
		var cs flight.CabinSection
		if err := sectionRows.Scan(&cs.Code, &cs.Name, &cs.Rows.From, &cs.Rows.To, &cs.Cols); err != nil {
			return nil, fmt.Errorf("scan section: %w", err)
		}
		sections = append(sections, cs)
	}

	return &flight.SeatMapData{
		Aircraft: acModel,
		Cabins:   sections,
		Seats:    seats,
		Legend: map[string]flight.LegendItem{
			"WINDOW": {Color: "#4f46e5", Label: "Window"},
			"AISLE":  {Color: "#f59e0b", Label: "Aisle"},
			"MIDDLE": {Color: "#6b7280", Label: "Middle"},
		},
	}, nil
}

func (r *Postgres) ListAirports(ctx context.Context, query string) ([]flight.Airport, error) {
	var rows *sql.Rows
	var err error
	if query != "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, code, name, city, country, timezone, status, created_at
			FROM airports WHERE code ILIKE $1 OR city ILIKE $1 OR name ILIKE $1
			ORDER BY code
		`, "%"+query+"%")
	} else {
		rows, err = r.db.QueryContext(ctx, `SELECT id, code, name, city, country, timezone, status, created_at FROM airports ORDER BY code`)
	}
	if err != nil {
		return nil, fmt.Errorf("list airports: %w", err)
	}
	defer rows.Close()

	var results []flight.Airport
	for rows.Next() {
		var a flight.Airport
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.City, &a.Country, &a.Timezone, &a.Status, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan airport: %w", err)
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

func (r *Postgres) ListAirlines(ctx context.Context) ([]flight.Airline, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, code, name, country, status, created_at FROM airlines ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("list airlines: %w", err)
	}
	defer rows.Close()
	var results []flight.Airline
	for rows.Next() {
		var a flight.Airline
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.Country, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

func (r *Postgres) ListAircraft(ctx context.Context) ([]flight.Aircraft, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, airline_id, model, registration, total_seats, created_at FROM aircraft ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list aircraft: %w", err)
	}
	defer rows.Close()
	var results []flight.Aircraft
	for rows.Next() {
		var a flight.Aircraft
		if err := rows.Scan(&a.ID, &a.AirlineID, &a.Model, &a.Registration, &a.TotalSeats, &a.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

func (r *Postgres) ListRoutes(ctx context.Context) ([]flight.Route, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.airline_id, r.origin_id, r.destination_id, r.distance_km, r.base_price, r.currency, r.status, r.created_at
		FROM routes r ORDER BY r.id
	`)
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	defer rows.Close()
	var results []flight.Route
	for rows.Next() {
		var rt flight.Route
		if err := rows.Scan(&rt.ID, &rt.AirlineID, &rt.OriginID, &rt.DestinationID, &rt.DistanceKM, &rt.BasePrice, &rt.Currency, &rt.Status, &rt.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, rt)
	}
	return results, rows.Err()
}

func (r *Postgres) ListFareClasses(ctx context.Context, flightInstanceID int64) ([]flight.FareAvailability, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT fc.id, fc.cabin_id, fc.code, fc.name, fc.baggage_allowance, fc.change_allowed, fc.change_penalty, fc.refund_allowed, fc.refund_penalty, fc.upgrade_allowed,
		       COALESCE(fic.available_count, 0)
		FROM fare_classes fc
		LEFT JOIN flight_instance_fare_classes fic ON fc.id = fic.fare_class_id AND fic.flight_instance_id = $1
		ORDER BY fc.id
	`, flightInstanceID)
	if err != nil {
		return nil, fmt.Errorf("list fare classes: %w", err)
	}
	defer rows.Close()
	var results []flight.FareAvailability
	for rows.Next() {
		var fa flight.FareAvailability
		if err := rows.Scan(&fa.ID, &fa.CabinID, &fa.Code, &fa.Name, &fa.BaggageAllowance, &fa.ChangeAllowed, &fa.ChangePenalty, &fa.RefundAllowed, &fa.RefundPenalty, &fa.UpgradeAllowed, &fa.Available); err != nil {
			return nil, err
		}
		results = append(results, fa)
	}
	return results, rows.Err()
}

func (r *Postgres) CreateRoute(ctx context.Context, in flight.RouteInput) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO routes (airline_id, origin_id, destination_id, distance_km, base_price, currency, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE') RETURNING id
	`, in.AirlineID, in.OriginID, in.DestinationID, in.DistanceKM, in.BasePrice, in.Currency).Scan(&id)
	return id, err
}

func (r *Postgres) CreateFlightSchedule(ctx context.Context, in flight.ScheduleInput) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO flight_schedules (route_id, aircraft_id, flight_number, departure_time, arrival_time, days_of_week, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE') RETURNING id
	`, in.RouteID, in.AircraftID, in.FlightNumber, in.DepartureTime, in.ArrivalTime, in.DaysOfWeek).Scan(&id)
	return id, err
}

func (r *Postgres) MaterializeSchedule(ctx context.Context, scheduleID int64, date time.Time) (int, error) {
	sched, err := r.GetFlightSchedule(ctx, scheduleID)
	if err != nil {
		return 0, err
	}
	if sched == nil {
		return 0, fmt.Errorf("schedule not found: %d", scheduleID)
	}

	var existing int64
	err = r.db.QueryRowContext(ctx, `SELECT id FROM flight_instances WHERE schedule_id = $1 AND departure_date = $2`, scheduleID, date).Scan(&existing)
	if err == nil {
		return 0, nil
	}

	var totalSeats int
	err = r.db.QueryRowContext(ctx, `SELECT total_seats FROM aircraft WHERE id = $1`, sched.AircraftID).Scan(&totalSeats)
	if err != nil {
		return 0, err
	}

	depTime := time.Date(date.Year(), date.Month(), date.Day(), sched.DepartureTime.Hour(), sched.DepartureTime.Minute(), 0, 0, time.UTC)
	arrTime := depTime.Add(time.Duration(sched.DaysOfWeek) * time.Minute)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO flight_instances (schedule_id, departure_date, departure_time, arrival_time, status)
		VALUES ($1, $2, $3, $4, 'SCHEDULED')
	`, scheduleID, date, depTime, arrTime)
	if err != nil {
		return 0, err
	}
	return 1, nil
}

func (r *Postgres) UpdateFlightInstance(ctx context.Context, id int64, f flight.FlightInstanceUpdate) error {
	query := "UPDATE flight_instances SET"
	args := []interface{}{}
	argCount := 1

	if f.DepartureTime != nil {
		query += fmt.Sprintf(" departure_time = $%d,", argCount)
		args = append(args, *f.DepartureTime)
		argCount++
	}
	if f.ArrivalTime != nil {
		query += fmt.Sprintf(" arrival_time = $%d,", argCount)
		args = append(args, *f.ArrivalTime)
		argCount++
	}
	if f.DelayMinutes != nil {
		query += fmt.Sprintf(" delay_minutes = $%d,", argCount)
		args = append(args, *f.DelayMinutes)
		argCount++
	}
	if f.Status != nil {
		query += fmt.Sprintf(" status = $%d,", argCount)
		args = append(args, *f.Status)
		argCount++
	}

	query += " updated_at = NOW()"
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Postgres) DeleteFlightInstance(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM flight_instances WHERE id = $1`, id)
	return err
}

func (r *Postgres) ListFlightInstances(ctx context.Context, query flight.ListQuery) ([]flight.FlightInstance, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, schedule_id, departure_date, departure_time, arrival_time, status, delay_minutes, created_at FROM flight_instances ORDER BY departure_time`)
	if err != nil {
		return nil, fmt.Errorf("list flight instances: %w", err)
	}
	defer rows.Close()
	var results []flight.FlightInstance
	for rows.Next() {
		var fi flight.FlightInstance
		if err := rows.Scan(&fi.ID, &fi.ScheduleID, &fi.DepartureDate, &fi.DepartureTime, &fi.ArrivalTime, &fi.Status, &fi.DelayMinutes, &fi.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, fi)
	}
	return results, rows.Err()
}

func (r *Postgres) ResolveFareClass(ctx context.Context, flightInstanceID int64, code string) (int64, int, error) {
	var id, available int64
	err := r.db.QueryRowContext(ctx, `
		SELECT fc.id, COALESCE(fic.available_count, 0)
		FROM fare_classes fc
		LEFT JOIN flight_instance_fare_classes fic ON fc.id = fic.fare_class_id AND fic.flight_instance_id = $1
		WHERE fc.code = $2
	`, flightInstanceID, code).Scan(&id, &available)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	return id, int(available), err
}

func (r *Postgres) FareClassCapacity(ctx context.Context, flightInstanceID, fareClassID int64) (int, error) {
	var capacity int
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(available_count, 0) FROM flight_instance_fare_classes
		WHERE flight_instance_id = $1 AND fare_class_id = $2
	`, flightInstanceID, fareClassID).Scan(&capacity)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return capacity, err
}

func (r *Postgres) AircraftSeatCount(ctx context.Context, flightInstanceID int64) (int, error) {
	var totalSeats int
	err := r.db.QueryRowContext(ctx, `
		SELECT ac.total_seats
		FROM flight_instances fi
		JOIN flight_schedules fs ON fi.schedule_id = fs.id
		JOIN aircraft ac ON fs.aircraft_id = ac.id
		WHERE fi.id = $1
	`, flightInstanceID).Scan(&totalSeats)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("flight instance not found: %d", flightInstanceID)
	}
	return totalSeats, err
}

func (r *Postgres) FlightInfo(ctx context.Context, flightInstanceID int64) (string, string, string, error) {
	var flightNumber, origin, destination string
	err := r.db.QueryRowContext(ctx, `
		SELECT fs.flight_number, orig.code, dest.code
		FROM flight_instances fi
		JOIN flight_schedules fs ON fi.schedule_id = fs.id
		JOIN routes r ON fs.route_id = r.id
		JOIN airports orig ON r.origin_id = orig.id
		JOIN airports dest ON r.destination_id = dest.id
		WHERE fi.id = $1
	`, flightInstanceID).Scan(&flightNumber, &origin, &destination)
	if err != nil {
		return "", "", "", err
	}
	return flightNumber, origin, destination, nil
}

// Helper methods
func (r *Postgres) cabinCode(ctx context.Context, cabinID int64) string {
	var code string
	_ = r.db.QueryRowContext(ctx, `SELECT code FROM cabin_classes WHERE id = $1`, cabinID).Scan(&code)
	return code
}

func (r *Postgres) fareClassBasePrice(code string) float64 {
	switch code {
	case "YSAVER": return 189
	case "YFLEX": return 349
	case "JSUPER": return 1299
	case "FFIRST": return 2499
	}
	return 300
}

func fareRound2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

var _ flight.Repository = (*Postgres)(nil)
