package pricing

import (
	"context"
	"testing"
)

func TestCalculateKnownFare(t *testing.T) {
	e := NewEngine(nil)
	q, err := e.Calculate(context.Background(), 1000, "YSAVER", 2)
	if err != nil {
		t.Fatal(err)
	}
	if q.FlightInstanceID != 1000 || q.FareClassCode != "YSAVER" || q.Passengers != 2 {
		t.Fatalf("unexpected quote: %+v", q)
	}
	// YSAVER base 189; taxes = 189*0.075 + 4.50 + 5.60; fees = 189*0.05.
	expectedPerPax := 189.00 + 189.00*0.075 + 4.50 + 5.60 + 189.00*0.05
	if got := q.TotalAmount; got != expectedPerPax*2 {
		t.Fatalf("total = %.2f, want %.2f", got, expectedPerPax*2)
	}
	if q.ExpiresAt.IsZero() {
		t.Fatal("expires_at should be set")
	}
}

func TestCalculateUnknownFareDefaults(t *testing.T) {
	e := NewEngine(nil)
	q, err := e.Calculate(context.Background(), 1, "NOPE", 1)
	if err != nil {
		t.Fatal(err)
	}
	if q.TotalAmount == 0 {
		t.Fatal("unknown fare class should fall back to a base price")
	}
}

func TestRules(t *testing.T) {
	e := NewEngine(nil)
	r, err := e.CreateRule("YSAVER", 250.0)
	if err != nil {
		t.Fatal(err)
	}
	if r.ID == 0 || r.BasePrice != 250.0 {
		t.Fatalf("created rule: %+v", r)
	}

	q, _ := e.Calculate(context.Background(), 1, "YSAVER", 1)
	if q.TotalAmount != 250.0+250.0*0.075+4.50+5.60+250.0*0.05 {
		t.Fatalf("rule override not applied: %.2f", q.TotalAmount)
	}

	// Duplicate fare-class code replaces the rule idempotently.
	r2, err := e.CreateRule("YSAVER", 300.0)
	if err != nil {
		t.Fatal(err)
	}
	if r2.ID != r.ID {
		t.Fatalf("expected idempotent update, got new id %d", r2.ID)
	}
	if len(e.ListRules()) != 1 {
		t.Fatalf("rules = %d, want 1", len(e.ListRules()))
	}

	if err := e.DeleteRule(r.ID); err != nil {
		t.Fatal(err)
	}
	if len(e.ListRules()) != 0 {
		t.Fatal("rule should be deleted")
	}
}

func TestCreateRuleValidation(t *testing.T) {
	e := NewEngine(nil)
	if _, err := e.CreateRule("", 100); err == nil {
		t.Fatal("empty code should fail")
	}
	if _, err := e.CreateRule("Y", -1); err == nil {
		t.Fatal("negative price should fail")
	}
}

func TestBreakdown(t *testing.T) {
	e := NewEngine(nil)
	b := e.GetBreakdown(100.0)
	if b.BaseFare != 100.0 {
		t.Fatalf("base = %.2f", b.BaseFare)
	}
	if len(b.Taxes) != 3 {
		t.Fatalf("taxes = %d, want 3", len(b.Taxes))
	}
	total := b.BaseFare
	for _, tx := range b.Taxes {
		total += tx.Amount
	}
	total += b.Fees
	if b.Total != total {
		t.Fatalf("total mismatch: %.2f vs %.2f", b.Total, total)
	}
}
