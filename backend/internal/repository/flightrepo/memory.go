package flightrepo

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/you/airline-ticketing/internal/domain/flight"
	"github.com/you/airline-ticketing/internal/pkg/seat"
)

var errNotFound = errors.New("flightrepo: not found")

// Memory is an in-memory flight.Repository seeded with demo data. It is
// the default backing store so the API runs without Postgres; swapping in
// a Postgres implementation keeps the same interface.
type Memory struct {
	mu          sync.RWMutex
	airports    map[int64]flight.Airport
	airlines    map[int64]flight.Airline
	aircraft    map[int64]flight.Aircraft
	routes      map[int64]flight.Route
	schedules   map[int64]flight.FlightSchedule
	instances   map[int64]flight.FlightInstance
	seats       map[int64][]flight.Seat
	sections    map[int64][]flight.CabinSection
	cabins      map[int64]flight.CabinClass
	fareClasses map[int64]flight.FareClass
	fares       map[int64][]flight.FareAvailability
	nextID      map[string]int64
}

func New() *Memory {
	m := &Memory{
		airports:    map[int64]flight.Airport{},
		airlines:    map[int64]flight.Airline{},
		aircraft:    map[int64]flight.Aircraft{},
		routes:      map[int64]flight.Route{},
		schedules:   map[int64]flight.FlightSchedule{},
		instances:   map[int64]flight.FlightInstance{},
		seats:       map[int64][]flight.Seat{},
		sections:    map[int64][]flight.CabinSection{},
		cabins:      map[int64]flight.CabinClass{},
		fareClasses: map[int64]flight.FareClass{},
		fares:       map[int64][]flight.FareAvailability{},
		nextID:      map[string]int64{},
	}
	m.seed()
	return m
}

// ---- seed data ----

func (m *Memory) seed() {
	now := time.Now().UTC().Truncate(24 * time.Hour)

	airports := []flight.Airport{
		{ID: 1, Code: "JFK", Name: "John F. Kennedy International Airport", City: "New York", Country: "US", Timezone: "America/New_York", Status: "ACTIVE", CreatedAt: now},
		{ID: 2, Code: "LAX", Name: "Los Angeles International Airport", City: "Los Angeles", Country: "US", Timezone: "America/Los_Angeles", Status: "ACTIVE", CreatedAt: now},
		{ID: 3, Code: "SFO", Name: "San Francisco International Airport", City: "San Francisco", Country: "US", Timezone: "America/Los_Angeles", Status: "ACTIVE", CreatedAt: now},
		{ID: 4, Code: "MIA", Name: "Miami International Airport", City: "Miami", Country: "US", Timezone: "America/New_York", Status: "ACTIVE", CreatedAt: now},
		{ID: 5, Code: "ORD", Name: "O'Hare International Airport", City: "Chicago", Country: "US", Timezone: "America/Chicago", Status: "ACTIVE", CreatedAt: now},
	}
	for _, a := range airports {
		m.airports[a.ID] = a
	}

	airlines := []flight.Airline{
		{ID: 1, Code: "AA", Name: "American Airlines", Country: "US", Status: "ACTIVE", CreatedAt: now},
		{ID: 2, Code: "UA", Name: "United Airlines", Country: "US", Status: "ACTIVE", CreatedAt: now},
		{ID: 3, Code: "DL", Name: "Delta Air Lines", Country: "US", Status: "ACTIVE", CreatedAt: now},
	}
	for _, a := range airlines {
		m.airlines[a.ID] = a
	}

	aircraft := []flight.Aircraft{
		{ID: 1, AirlineID: 1, Model: "Airbus A320neo", Registration: "N101AA", TotalSeats: 176, CreatedAt: now},
		{ID: 2, AirlineID: 2, Model: "Boeing 787-9", Registration: "N202UA", TotalSeats: 252, CreatedAt: now},
		{ID: 3, AirlineID: 3, Model: "Airbus A321neo", Registration: "N303DL", TotalSeats: 196, CreatedAt: now},
	}
	for _, a := range aircraft {
		m.aircraft[a.ID] = a
		m.seats[a.ID] = m.buildSeats(a.ID)
		m.sections[a.ID] = m.buildSections(a.ID)
	}

	cabins := []flight.CabinClass{
		{ID: 1, Code: "F", Name: "First"},
		{ID: 2, Code: "J", Name: "Business"},
		{ID: 3, Code: "Y", Name: "Economy"},
		{ID: 4, Code: "W", Name: "Premium Economy"},
	}
	for _, c := range cabins {
		m.cabins[c.ID] = c
	}

	fareClasses := []flight.FareClass{
		{ID: 1, CabinID: 1, Code: "FFIRST", Name: "First Flexible", ChangeAllowed: true, RefundAllowed: true, UpgradeAllowed: false},
		{ID: 2, CabinID: 2, Code: "JSUPER", Name: "Business Super", ChangeAllowed: true, RefundAllowed: true, UpgradeAllowed: false},
		{ID: 3, CabinID: 3, Code: "YSAVER", Name: "Economy Saver", ChangeAllowed: false, RefundAllowed: false, UpgradeAllowed: true},
		{ID: 4, CabinID: 3, Code: "YFLEX", Name: "Economy Flexible", ChangeAllowed: true, RefundAllowed: true, UpgradeAllowed: true},
	}
	for _, f := range fareClasses {
		m.fareClasses[f.ID] = f
	}

	routes := []flight.Route{
		{ID: 1, AirlineID: 1, OriginID: 1, DestinationID: 2, BasePrice: 189, Currency: "USD", Status: "ACTIVE", CreatedAt: now},
		{ID: 2, AirlineID: 1, OriginID: 1, DestinationID: 4, BasePrice: 159, Currency: "USD", Status: "ACTIVE", CreatedAt: now},
		{ID: 3, AirlineID: 2, OriginID: 3, DestinationID: 1, BasePrice: 209, Currency: "USD", Status: "ACTIVE", CreatedAt: now},
		{ID: 4, AirlineID: 2, OriginID: 2, DestinationID: 3, BasePrice: 119, Currency: "USD", Status: "ACTIVE", CreatedAt: now},
		{ID: 5, AirlineID: 3, OriginID: 5, DestinationID: 4, BasePrice: 139, Currency: "USD", Status: "ACTIVE", CreatedAt: now},
		{ID: 6, AirlineID: 3, OriginID: 1, DestinationID: 3, BasePrice: 199, Currency: "USD", Status: "ACTIVE", CreatedAt: now},
	}
	for _, r := range routes {
		m.routes[r.ID] = r
	}

	schedules := []flight.FlightSchedule{
		{ID: 1, RouteID: 1, AircraftID: 1, FlightNumber: "AA101", DepartureTime: t(now, 6, 30), ArrivalTime: t(now, 9, 45), DaysOfWeek: 127, Status: "ACTIVE", CreatedAt: now},
		{ID: 2, RouteID: 2, AircraftID: 3, FlightNumber: "AA155", DepartureTime: t(now, 8, 10), ArrivalTime: t(now, 11, 30), DaysOfWeek: 127, Status: "ACTIVE", CreatedAt: now},
		{ID: 3, RouteID: 3, AircraftID: 2, FlightNumber: "UA200", DepartureTime: t(now, 7, 0), ArrivalTime: t(now, 15, 20), DaysOfWeek: 127, Status: "ACTIVE", CreatedAt: now},
		{ID: 4, RouteID: 4, AircraftID: 2, FlightNumber: "UA310", DepartureTime: t(now, 10, 30), ArrivalTime: t(now, 11, 55), DaysOfWeek: 127, Status: "ACTIVE", CreatedAt: now},
		{ID: 5, RouteID: 5, AircraftID: 3, FlightNumber: "DL500", DepartureTime: t(now, 9, 15), ArrivalTime: t(now, 12, 40), DaysOfWeek: 127, Status: "ACTIVE", CreatedAt: now},
		{ID: 6, RouteID: 6, AircraftID: 2, FlightNumber: "DL600", DepartureTime: t(now, 13, 0), ArrivalTime: t(now, 15, 40), DaysOfWeek: 127, Status: "ACTIVE", CreatedAt: now},
	}
	m.nextID["airport"] = 100
	m.nextID["airline"] = 100
	m.nextID["aircraft"] = 100
	m.nextID["route"] = 100
	m.nextID["schedule"] = 100
	m.nextID["instance"] = 1000

	for _, s := range schedules {
		m.schedules[s.ID] = s
		m.materializeLocked(s.ID, now)
	}
}

func t(day time.Time, h, min int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), h, min, 0, 0, time.UTC)
}

// ---- layout helpers ----

type cabinLayout struct {
	code string
	name string
	from int
	to   int
	cols []string
}

func layoutFor(aircraftID int64) []cabinLayout {
	switch aircraftID {
	case 2: // 787-9
		return []cabinLayout{
			{code: "F", name: "First", from: 1, to: 5, cols: []string{"A", "B", "C", "D"}},
			{code: "J", name: "Business", from: 6, to: 16, cols: []string{"A", "B", "C", "D", "E", "F"}},
			{code: "Y", name: "Economy", from: 17, to: 42, cols: []string{"A", "B", "C", "D", "E", "F"}},
		}
	default:
		return []cabinLayout{
			{code: "F", name: "First", from: 1, to: 4, cols: []string{"A", "B", "C", "D"}},
			{code: "J", name: "Business", from: 5, to: 12, cols: []string{"A", "B", "C", "D", "E", "F"}},
			{code: "Y", name: "Economy", from: 13, to: 38, cols: []string{"A", "B", "C", "D", "E", "F"}},
		}
	}
}

func cabinID(code string) int64 {
	switch code {
	case "F":
		return 1
	case "J":
		return 2
	case "Y":
		return 3
	case "W":
		return 4
	}
	return 3
}

func (m *Memory) buildSeats(aircraftID int64) []flight.Seat {
	var seats []flight.Seat
	for _, cabin := range layoutFor(aircraftID) {
		emergency := map[int]bool{}
		for _, r := range []int{cabin.from + 1, cabin.to} {
			emergency[r] = true
		}
		for row := cabin.from; row <= cabin.to; row++ {
			for _, col := range cabin.cols {
				seats = append(seats, flight.Seat{
					ID:              seat.EncodeID(row, col),
					AircraftID:      aircraftID,
					RowNum:          row,
					Col:             col,
					Position:        seatPosition(cabin.cols, col),
					CabinClassID:    cabinID(cabin.code),
					IsEmergencyExit: emergency[row],
					IsExtraLegroom:  row <= cabin.from+2 && cabin.code != "Y",
				})
			}
		}
	}
	return seats
}

func (m *Memory) buildSections(aircraftID int64) []flight.CabinSection {
	var sections []flight.CabinSection
	for _, cabin := range layoutFor(aircraftID) {
		sections = append(sections, flight.CabinSection{
			Code: cabin.code,
			Name: cabin.name,
			Rows: flight.RowRange{From: cabin.from, To: cabin.to},
			Cols: cabin.cols,
		})
	}
	return sections
}

func seatPosition(cols []string, col string) string {
	last := cols[len(cols)-1]
	switch {
	case col == cols[0] || col == last:
		return "WINDOW"
	case len(cols) <= 4 && (col == "B" || col == "C"):
		return "AISLE"
	case len(cols) >= 5 && (col == "C" || col == "D"):
		return "AISLE"
	default:
		return "MIDDLE"
	}
}

func (m *Memory) cabinSeatCount(aircraftID int64, cabinClassID int64) int {
	count := 0
	for _, s := range m.seats[aircraftID] {
		if s.CabinClassID == cabinClassID {
			count++
		}
	}
	return count
}

func (m *Memory) fareClassBasePrice(code string) float64 {
	switch code {
	case "YSAVER":
		return 189
	case "YFLEX":
		return 349
	case "JSUPER":
		return 1299
	case "FFIRST":
		return 2499
	}
	return 300
}

func (m *Memory) buildFaresLocked(instanceID, aircraftID int64) {
	var fares []flight.FareAvailability
	for id := int64(1); id <= 4; id++ {
		fc := m.fareClasses[id]
		if _, ok := m.cabins[fc.CabinID]; !ok {
			continue
		}
		fares = append(fares, flight.FareAvailability{
			FareClass: fc,
			Available: m.cabinSeatCount(aircraftID, fc.CabinID),
		})
	}
	m.fares[instanceID] = fares
}

func (m *Memory) materializeLocked(scheduleID int64, day time.Time) {
	sched, ok := m.schedules[scheduleID]
	if !ok {
		return
	}
	dep := t(day, sched.DepartureTime.Hour(), sched.DepartureTime.Minute())
	arr := t(day, sched.ArrivalTime.Hour(), sched.ArrivalTime.Minute())
	id := m.nextID["instance"]
	m.nextID["instance"] = id + 1
	m.instances[id] = flight.FlightInstance{
		ID:            id,
		ScheduleID:    scheduleID,
		DepartureDate: day,
		DepartureTime: dep,
		ArrivalTime:   arr,
		Status:        "SCHEDULED",
		CreatedAt:     time.Now().UTC(),
	}
	m.buildFaresLocked(id, sched.AircraftID)
}

// ---- Repository implementation ----

func (m *Memory) Search(ctx context.Context, criteria flight.SearchCriteria) ([]flight.SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []flight.SearchResult
	for _, inst := range m.instances {
		sched, ok := m.schedules[inst.ScheduleID]
		if !ok {
			continue
		}
		route, ok := m.routes[sched.RouteID]
		if !ok {
			continue
		}
		origin, ok := m.airports[route.OriginID]
		if !ok {
			continue
		}
		dest, ok := m.airports[route.DestinationID]
		if !ok {
			continue
		}
		if !strings.EqualFold(string(origin.Code), string(criteria.Origin)) ||
			!strings.EqualFold(string(dest.Code), string(criteria.Destination)) {
			continue
		}
		if !criteria.Date.IsZero() && !sameDay(criteria.Date, inst.DepartureDate) {
			continue
		}

		airline := m.airlines[route.AirlineID]
		ac := m.aircraft[sched.AircraftID]

		var fares []flight.FareOption
		for _, fa := range m.fares[inst.ID] {
			cabinCode := m.cabins[fa.CabinID].Code
			if criteria.CabinClass != "" && !strings.EqualFold(cabinCode, criteria.CabinClass) {
				continue
			}
			base := m.fareClassBasePrice(fa.Code)
			taxes := base*0.075 + 4.50 + 5.60
			fees := base * 0.05
			fares = append(fares, flight.FareOption{
				FareClassCode: fa.Code,
				Cabin:         cabinCode,
				Available:     fa.Available,
				Price:         round2(base),
				Taxes:         round2(taxes),
				Total:         round2(base + taxes + fees),
			})
		}
		if len(fares) == 0 {
			continue
		}

		results = append(results, flight.SearchResult{
			FlightID:     flight.ID(inst.ID),
			Airline:      airline,
			FlightNumber: sched.FlightNumber,
			Origin:       origin,
			Destination:  dest,
			Departure:    inst.DepartureTime,
			Arrival:      inst.ArrivalTime,
			DurationMins: int(inst.ArrivalTime.Sub(inst.DepartureTime).Minutes()),
			Aircraft:     ac.Model,
			Fares:        fares,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Departure.Before(results[j].Departure)
	})
	return results, nil
}

func (m *Memory) GetFlightInstance(ctx context.Context, id int64) (*flight.FlightInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[id]
	if !ok {
		return nil, nil
	}
	cp := inst
	return &cp, nil
}

func (m *Memory) GetFlightSchedule(ctx context.Context, id int64) (*flight.FlightSchedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sched, ok := m.schedules[id]
	if !ok {
		return nil, nil
	}
	cp := sched
	return &cp, nil
}

func (m *Memory) GetSeatMap(ctx context.Context, flightInstanceID int64) (*flight.SeatMapData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[flightInstanceID]
	if !ok {
		return nil, errNotFound
	}
	sched := m.schedules[inst.ScheduleID]
	ac := m.aircraft[sched.AircraftID]

	var seats []flight.SeatInfo
	for _, s := range m.seats[ac.ID] {
		seats = append(seats, flight.SeatInfo{
			ID:              s.ID,
			Row:             s.RowNum,
			Col:             s.Col,
			Position:        s.Position,
			Cabin:           m.cabins[s.CabinClassID].Code,
			IsAvailable:     true,
			IsEmergencyExit: s.IsEmergencyExit,
			IsExtraLegroom:  s.IsExtraLegroom,
		})
	}
	sections := append([]flight.CabinSection(nil), m.sections[ac.ID]...)
	return &flight.SeatMapData{
		Aircraft: ac.Model,
		Cabins:   sections,
		Seats:    seats,
		Legend: map[string]flight.LegendItem{
			"WINDOW": {Color: "#4f46e5", Label: "Window"},
			"AISLE":  {Color: "#f59e0b", Label: "Aisle"},
			"MIDDLE": {Color: "#6b7280", Label: "Middle"},
		},
	}, nil
}

func (m *Memory) ListAirports(ctx context.Context, query string) ([]flight.Airport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(query))
	var out []flight.Airport
	for _, a := range m.airports {
		if q == "" || strings.Contains(strings.ToLower(string(a.Code)+" "+a.City+" "+a.Name), q) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (m *Memory) ListAirlines(ctx context.Context) ([]flight.Airline, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []flight.Airline
	for _, a := range m.airlines {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out, nil
}

func (m *Memory) ListAircraft(ctx context.Context) ([]flight.Aircraft, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []flight.Aircraft
	for _, a := range m.aircraft {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *Memory) ListRoutes(ctx context.Context) ([]flight.Route, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []flight.Route
	for _, r := range m.routes {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *Memory) ListFareClasses(ctx context.Context, flightInstanceID int64) ([]flight.FareAvailability, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := append([]flight.FareAvailability(nil), m.fares[flightInstanceID]...)
	return out, nil
}

func (m *Memory) CreateRoute(ctx context.Context, in flight.RouteInput) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.airlines[in.AirlineID]; !ok {
		return 0, errors.New("unknown airline")
	}
	if _, ok := m.airports[in.OriginID]; !ok {
		return 0, errors.New("unknown origin airport")
	}
	if _, ok := m.airports[in.DestinationID]; !ok {
		return 0, errors.New("unknown destination airport")
	}
	id := m.nextID["route"]
	m.nextID["route"] = id + 1
	m.routes[id] = flight.Route{
		ID:            id,
		AirlineID:     in.AirlineID,
		OriginID:      in.OriginID,
		DestinationID: in.DestinationID,
		DistanceKM:    in.DistanceKM,
		BasePrice:     in.BasePrice,
		Currency:      in.Currency,
		Status:        "ACTIVE",
		CreatedAt:     time.Now().UTC(),
	}
	return id, nil
}

func (m *Memory) CreateFlightSchedule(ctx context.Context, in flight.ScheduleInput) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.routes[in.RouteID]; !ok {
		return 0, errors.New("unknown route")
	}
	if _, ok := m.aircraft[in.AircraftID]; !ok {
		return 0, errors.New("unknown aircraft")
	}
	id := m.nextID["schedule"]
	m.nextID["schedule"] = id + 1
	m.schedules[id] = flight.FlightSchedule{
		ID:            id,
		RouteID:       in.RouteID,
		AircraftID:    in.AircraftID,
		FlightNumber:  in.FlightNumber,
		DepartureTime: in.DepartureTime,
		ArrivalTime:   in.ArrivalTime,
		DaysOfWeek:    in.DaysOfWeek,
		Status:        "ACTIVE",
		CreatedAt:     time.Now().UTC(),
	}
	// Auto-materialize the next 14 days so the schedule is searchable.
	start := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 0; i < 14; i++ {
		m.materializeLocked(id, start.AddDate(0, 0, i))
	}
	return id, nil
}

func (m *Memory) MaterializeSchedule(ctx context.Context, scheduleID int64, date time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.schedules[scheduleID]; !ok {
		return 0, errors.New("unknown schedule")
	}
	day := date.UTC().Truncate(24 * time.Hour)
	for _, inst := range m.instances {
		if inst.ScheduleID == scheduleID && sameDay(inst.DepartureDate, day) {
			return 0, nil
		}
	}
	m.materializeLocked(scheduleID, day)
	return 1, nil
}

func (m *Memory) UpdateFlightInstance(ctx context.Context, id int64, f flight.FlightInstanceUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst, ok := m.instances[id]
	if !ok {
		return errNotFound
	}
	if f.DepartureTime != nil {
		inst.DepartureTime = *f.DepartureTime
	}
	if f.ArrivalTime != nil {
		inst.ArrivalTime = *f.ArrivalTime
	}
	if f.DelayMinutes != nil {
		inst.DelayMinutes = *f.DelayMinutes
	}
	if f.Status != nil {
		inst.Status = *f.Status
	}
	m.instances[id] = inst
	return nil
}

func (m *Memory) DeleteFlightInstance(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.instances[id]; !ok {
		return errNotFound
	}
	delete(m.instances, id)
	delete(m.fares, id)
	return nil
}

func (m *Memory) ListFlightInstances(ctx context.Context, query flight.ListQuery) ([]flight.FlightInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []flight.FlightInstance
	for _, inst := range m.instances {
		out = append(out, inst)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DepartureTime.Before(out[j].DepartureTime) })
	if query.Limit > 0 && len(out) > query.Limit {
		out = out[query.Offset:]
		if len(out) > query.Limit {
			out = out[:query.Limit]
		}
	}
	return out, nil
}

func (m *Memory) ResolveFareClass(ctx context.Context, flightInstanceID int64, code string) (int64, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, fa := range m.fares[flightInstanceID] {
		if fa.Code == code {
			return fa.ID, fa.Available, nil
		}
	}
	return 0, 0, nil
}

func (m *Memory) FareClassCapacity(ctx context.Context, flightInstanceID, fareClassID int64) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, fa := range m.fares[flightInstanceID] {
		if fa.ID == fareClassID {
			return fa.Available, nil
		}
	}
	return 0, nil
}

func (m *Memory) AircraftSeatCount(ctx context.Context, flightInstanceID int64) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[flightInstanceID]
	if !ok {
		return 0, errNotFound
	}
	sched := m.schedules[inst.ScheduleID]
	ac := m.aircraft[sched.AircraftID]
	return ac.TotalSeats, nil
}

func (m *Memory) FlightInfo(ctx context.Context, flightInstanceID int64) (string, string, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[flightInstanceID]
	if !ok {
		return "", "", "", errNotFound
	}
	sched := m.schedules[inst.ScheduleID]
	route := m.routes[sched.RouteID]
	origin := m.airports[route.OriginID]
	dest := m.airports[route.DestinationID]
	return sched.FlightNumber, string(origin.Code), string(dest.Code), nil
}

// ---- helpers ----

func sameDay(a, b time.Time) bool {
	ya, ma, da := a.Date()
	yb, mb, db := b.Date()
	return ya == yb && ma == mb && da == db
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
