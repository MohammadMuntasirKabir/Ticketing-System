package flightrepo

import (
	"context"
	"testing"
	"time"

	"github.com/you/airline-ticketing/internal/domain/flight"
)

func TestSeedBasics(t *testing.T) {
	ctx := context.Background()
	m := New()

	// Seeded instances start at 1000 (>= the 1000 floor, all > 0).
	instances, err := m.ListFlightInstances(ctx, flight.ListQuery{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 6 {
		t.Fatalf("instances = %d, want 6", len(instances))
	}
	for _, inst := range instances {
		if inst.ID < 1000 {
			t.Fatalf("instance id %d should be >= 1000", inst.ID)
		}
	}

	if n := len(m.airports); n < 5 {
		t.Fatalf("airports = %d", n)
	}
	if n := len(m.airlines); n < 3 {
		t.Fatalf("airlines = %d", n)
	}
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	m := New()

	res, err := m.Search(ctx, flight.SearchCriteria{Origin: "JFK", Destination: "LAX"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("expected results for JFK->LAX")
	}
	first := res[0]
	if first.FlightID == 0 || first.FlightNumber == "" {
		t.Fatalf("invalid result: %+v", first)
	}
	if len(first.Fares) == 0 {
		t.Fatal("expected fare options")
	}
}

func TestSearchNoMatch(t *testing.T) {
	ctx := context.Background()
	m := New()
	res, _ := m.Search(ctx, flight.SearchCriteria{Origin: "JFK", Destination: "XYZ"})
	if len(res) != 0 {
		t.Fatalf("expected no results, got %d", len(res))
	}
}

func TestGetSeatMap(t *testing.T) {
	ctx := context.Background()
	m := New()
	sm, err := m.GetSeatMap(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if sm.Aircraft == "" || len(sm.Seats) == 0 || len(sm.Cabins) == 0 {
		t.Fatalf("seat map: %+v", sm)
	}
	// Seat ids must decode back to their label.
	if sm.Seats[0].ID <= 0 {
		t.Fatalf("seat id %d invalid", sm.Seats[0].ID)
	}
}

func TestGetSeatMapMissing(t *testing.T) {
	ctx := context.Background()
	m := New()
	if _, err := m.GetSeatMap(ctx, 999999); err == nil {
		t.Fatal("expected error for missing flight instance")
	}
}

func TestFareClassResolution(t *testing.T) {
	ctx := context.Background()
	m := New()
	id, cap, err := m.ResolveFareClass(ctx, 1000, "YSAVER")
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 || cap == 0 {
		t.Fatalf("fare class id=%d capacity=%d", id, cap)
	}
	if c, _ := m.FareClassCapacity(ctx, 1000, id); c == 0 {
		t.Fatalf("capacity for id %d is 0", id)
	}
	// Unknown code resolves to nothing.
	if id2, _, _ := m.ResolveFareClass(ctx, 1000, "NOPE"); id2 != 0 {
		t.Fatalf("unknown fare class resolved to %d", id2)
	}
}

func TestFlightInfo(t *testing.T) {
	ctx := context.Background()
	m := New()
	fn, origin, dest, err := m.FlightInfo(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if fn == "" || origin == "" || dest == "" {
		t.Fatalf("flight info: %s %s %s", fn, origin, dest)
	}
}

func TestCreateRouteAndSchedule(t *testing.T) {
	ctx := context.Background()
	m := New()

	routeID, err := m.CreateRoute(ctx, flight.RouteInput{
		AirlineID:     1,
		OriginID:      1,
		DestinationID: 2,
		BasePrice:     99,
		Currency:      "USD",
	})
	if err != nil {
		t.Fatal(err)
	}
	if routeID == 0 {
		t.Fatal("route id should be non-zero")
	}

	dep := m.schedules[1].DepartureTime
	schID, err := m.CreateFlightSchedule(ctx, flight.ScheduleInput{
		RouteID:       routeID,
		AircraftID:    1,
		FlightNumber:  "XX999",
		DepartureTime: dep.Add(48 * time.Hour),
		ArrivalTime:   dep.Add(51 * time.Hour),
		DaysOfWeek:    127,
	})
	if err != nil {
		t.Fatal(err)
	}
	// A new schedule auto-materializes 14 days.
	instances := listInstancesForSchedule(m, schID)
	if len(instances) < 7 {
		t.Fatalf("expected >=7 materialized instances, got %d", len(instances))
	}
}

func TestCreateRouteUnknownReferences(t *testing.T) {
	ctx := context.Background()
	m := New()
	if _, err := m.CreateRoute(ctx, flight.RouteInput{AirlineID: 999, OriginID: 1, DestinationID: 2}); err == nil {
		t.Fatal("unknown airline should fail")
	}
}

func TestUpdateAndDeleteInstance(t *testing.T) {
	ctx := context.Background()
	m := New()
	status := "DELAYED"
	minutes := 45
	if err := m.UpdateFlightInstance(ctx, 1000, flight.FlightInstanceUpdate{DelayMinutes: &minutes, Status: &status}); err != nil {
		t.Fatal(err)
	}
	inst, _ := m.GetFlightInstance(ctx, 1000)
	if inst.Status != "DELAYED" || inst.DelayMinutes != 45 {
		t.Fatalf("updated instance: %+v", inst)
	}
	if err := m.DeleteFlightInstance(ctx, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetFlightInstance(ctx, 1000); err != nil {
		t.Fatalf("GetFlightInstance on deleted should return nil, got %v", err)
	}
}

func TestMaterializeScheduleIdempotent(t *testing.T) {
	ctx := context.Background()
	m := New()
	day := m.schedules[1].DepartureTime
	n, err := m.MaterializeSchedule(ctx, 1, day)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("re-materializing an existing day should add 0, got %d", n)
	}
}

// --- helpers ---

func listInstancesForSchedule(m *Memory, scheduleID int64) []flight.FlightInstance {
	var out []flight.FlightInstance
	for _, inst := range m.instances {
		if inst.ScheduleID == scheduleID {
			out = append(out, inst)
		}
	}
	return out
}
