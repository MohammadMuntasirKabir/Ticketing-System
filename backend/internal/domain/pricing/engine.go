package pricing

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	errEmptyCode     = errors.New("fare class code is required")
	errNegativePrice = errors.New("base price cannot be negative")
)

// Rule overrides the built-in base price for a fare-class code. Rules
// are served by the admin pricing endpoints; in-memory for now.
type Rule struct {
	ID            int64   `json:"id"`
	FareClassCode string  `json:"fare_class_code"`
	BasePrice     float64 `json:"base_price"`
}

type Engine struct {
	logger *zap.Logger
	mu     sync.RWMutex
	rules  []Rule
	nextID int64
}

func NewEngine(logger *zap.Logger) *Engine {
	return &Engine{logger: logger, nextID: 1}
}

type PriceBreakdown struct {
	BaseFare float64   `json:"base_fare"`
	Taxes    []TaxItem `json:"taxes"`
	Fees     float64   `json:"fees"`
	Total    float64   `json:"total"`
}

type TaxItem struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}

type QuoteResult struct {
	FlightInstanceID int64     `json:"flight_instance_id"`
	FareClassCode    string    `json:"fare_class_code"`
	Passengers       int       `json:"passengers"`
	TotalAmount      float64   `json:"total_amount"`
	ExpiresAt        time.Time `json:"expires_at"`
}

func (e *Engine) Calculate(ctx context.Context, flightInstID int64, fareClassCode string, passengers int) (*QuoteResult, error) {
	basePrice := e.calculateBasePrice(fareClassCode, flightInstID)
	taxes := e.calculateTaxes(basePrice)
	fees := e.calculateFees(basePrice)

	totalTaxes := 0.0
	for _, t := range taxes {
		totalTaxes += t.Amount
	}

	totalPerPax := basePrice + totalTaxes + fees
	grandTotal := totalPerPax * float64(passengers)

	return &QuoteResult{
		FlightInstanceID: flightInstID,
		FareClassCode:    fareClassCode,
		Passengers:       passengers,
		TotalAmount:      grandTotal,
		ExpiresAt:        time.Now().Add(15 * time.Minute),
	}, nil
}

func (e *Engine) calculateBasePrice(fareClassCode string, flightInstanceID int64) float64 {
	basePrices := map[string]float64{
		"YSAVER": 189.00,
		"YFLEX":  349.00,
		"JSUPER": 1299.00,
		"FFIRST": 2499.00,
	}

	base, ok := basePrices[fareClassCode]
	if !ok {
		base = 300.00
	}
	if price, found := e.rulePrice(fareClassCode); found {
		base = price
	}

	// Load-factor surcharging is intentionally not applied here: a real
	// implementation would source the sold ratio from inventory. Keeping
	// the multiplier at 1.0 makes pricing deterministic and testable.
	return base
}

// rulePrice returns a configured override for a fare-class code, if any.
func (e *Engine) rulePrice(fareClassCode string) (float64, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, r := range e.rules {
		if r.FareClassCode == fareClassCode {
			return r.BasePrice, true
		}
	}
	return 0, false
}

// ListRules returns the configured pricing rules.
func (e *Engine) ListRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := append([]Rule(nil), e.rules...)
	return out
}

// CreateRule adds a pricing rule. A duplicate fare-class code replaces
// the existing rule so admin edits are idempotent.
func (e *Engine) CreateRule(fareClassCode string, basePrice float64) (Rule, error) {
	if fareClassCode == "" {
		return Rule{}, errEmptyCode
	}
	if basePrice < 0 {
		return Rule{}, errNegativePrice
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.rules {
		if e.rules[i].FareClassCode == fareClassCode {
			e.rules[i].BasePrice = basePrice
			return e.rules[i], nil
		}
	}
	r := Rule{ID: e.nextID, FareClassCode: fareClassCode, BasePrice: basePrice}
	e.nextID++
	e.rules = append(e.rules, r)
	return r, nil
}

// DeleteRule removes a pricing rule by id. It is a no-op when the id is
// not present.
func (e *Engine) DeleteRule(id int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, r := range e.rules {
		if r.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return nil
		}
	}
	return nil
}

func (e *Engine) calculateTaxes(basePrice float64) []TaxItem {
	taxes := []TaxItem{}
	taxes = append(taxes, TaxItem{Type: "US Transportation Tax", Amount: basePrice * 0.075})
	taxes = append(taxes, TaxItem{Type: "Passenger Facility Charge", Amount: 4.50})
	taxes = append(taxes, TaxItem{Type: "September 11th Security Fee", Amount: 5.60})
	return taxes
}

func (e *Engine) calculateFees(basePrice float64) float64 {
	return basePrice * 0.05
}

func (e *Engine) GetBreakdown(basePrice float64) PriceBreakdown {
	taxes := e.calculateTaxes(basePrice)
	fees := e.calculateFees(basePrice)

	totalTaxes := 0.0
	for _, t := range taxes {
		totalTaxes += t.Amount
	}

	return PriceBreakdown{
		BaseFare: basePrice,
		Taxes:    taxes,
		Fees:     fees,
		Total:    basePrice + totalTaxes + fees,
	}
}
