package booking

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/domain/inventory"
	"github.com/you/airline-ticketing/internal/domain/payment"
	"github.com/you/airline-ticketing/internal/domain/pricing"
	"github.com/you/airline-ticketing/internal/pkg/contextkeys"
	"github.com/you/airline-ticketing/internal/pkg/errors"
	"github.com/you/airline-ticketing/internal/pkg/seat"
	"github.com/you/airline-ticketing/internal/pkg/store"
)

const (
	quoteTTL   = 15 * time.Minute
	pnrTTL     = 15 * time.Minute
	idemTTL    = 24 * time.Hour
	pnrCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// Orchestrator coordinates the booking saga: pricing -> inventory hold ->
// payment intent -> durable booking record -> PNR cache. It treats Redis
// (via store.KV) as the home for short-lived quote/PNR/idempotency state
// and the booking Repository as the source of truth for records.
type Orchestrator struct {
	flights  FareClassProvider
	inv      *inventory.Service
	pricing  *pricing.Engine
	payments payment.Provider
	bookings Repository
	kv       store.KV
	notify   Notifier
	logger   *zap.Logger
}

func NewOrchestrator(
	flights FareClassProvider,
	inv *inventory.Service,
	pricing *pricing.Engine,
	payments payment.Provider,
	bookings Repository,
	kv store.KV,
	notify Notifier,
	logger *zap.Logger,
) *Orchestrator {
	return &Orchestrator{
		flights:  flights,
		inv:      inv,
		pricing:  pricing,
		payments: payments,
		bookings: bookings,
		kv:       kv,
		notify:   notify,
		logger:   logger,
	}
}

// Notifier is the slice of notifications the orchestrator triggers.
type Notifier interface {
	SendBookingConfirmation(ctx context.Context, pnrID int64) error
}

// ---- Request / response types (JSON-serialisable transport shape) ----

type PriceQuoteRequest struct {
	FlightInstanceID int64  `json:"flight_instance_id" validate:"required,gt=0"`
	FareClassCode    string `json:"fare_class_code" validate:"required"`
	Passengers       int    `json:"passengers" validate:"required,min=1,max=9"`
}

type PriceQuoteResponse struct {
	QuoteID       string                 `json:"quote_id"`
	FlightID      int64                  `json:"flight_id"`
	FareClassCode string                 `json:"fare_class_code"`
	Price         float64                `json:"price"`
	Taxes         float64                `json:"taxes"`
	Total         float64                `json:"total"`
	Breakdown     pricing.PriceBreakdown `json:"breakdown"`
	Currency      string                 `json:"currency"`
	ExpiresAt     time.Time              `json:"expires_at"`
}

type PassengerInput struct {
	Title          string `json:"title" validate:"required"`
	FirstName      string `json:"first_name" validate:"required"`
	LastName       string `json:"last_name" validate:"required"`
	DateOfBirth    string `json:"date_of_birth" validate:"required"`
	Nationality    string `json:"nationality" validate:"required,len=2"`
	PassportNumber string `json:"passport_number" validate:"required"`
	PassportExpiry string `json:"passport_expiry" validate:"required"`
}

type CreateBookingRequest struct {
	QuoteID        string           `json:"quote_id" validate:"required"`
	IdempotencyKey string           `json:"idempotency_key"`
	Passengers     []PassengerInput `json:"passengers" validate:"required,min=1,dive"`
}

type CreateBookingResponse struct {
	PNR                 string    `json:"pnr"`
	PNRID               int64     `json:"pnr_id"`
	Status              string    `json:"status"`
	PaymentIntentID     string    `json:"payment_intent_id"`
	PaymentClientSecret string    `json:"payment_client_secret"`
	AmountCents         int64     `json:"amount_cents"`
	Currency            string    `json:"currency"`
	HoldExpiresAt       time.Time `json:"hold_expires_at"`
}

type ConfirmResponse struct {
	PNR    string `json:"pnr"`
	Status string `json:"status"`
}

type CancelResponse struct {
	PNR    string `json:"pnr"`
	Status string `json:"status"`
}

type SeatAssignmentRequest struct {
	SeatNumber string `json:"seat_number" validate:"required"`
}

type SeatAssignmentResponse struct {
	BookingID int64  `json:"booking_id"`
	SeatID    *int64 `json:"seat_id"`
	Seat      string `json:"seat"`
}

type BookingDetail struct {
	PNR        string               `json:"pnr"`
	Status     string               `json:"status"`
	TotalCents int64                `json:"total_cents"`
	Currency   string               `json:"currency"`
	CreatedAt  time.Time            `json:"created_at"`
	Bookings   []BookingDetailEntry `json:"bookings"`
	Passengers []PassengerDetail    `json:"passengers"`
}

type BookingDetailEntry struct {
	ID           int64  `json:"id"`
	FlightNumber string `json:"flight_number"`
	FareClass    string `json:"fare_class"`
	Seat         string `json:"seat"`
	Status       string `json:"status"`
}

type PassengerDetail struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	TicketNumber string `json:"ticket_number"`
}

type PriceQuote struct {
	FlightInstanceID int64     `json:"flight_instance_id"`
	FareClassCode    string    `json:"fare_class_code"`
	Passengers       int       `json:"passengers"`
	TotalAmount      float64   `json:"total_amount"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// ---- Use cases ----

func (o *Orchestrator) GetPriceQuote(ctx context.Context, req PriceQuoteRequest) (*PriceQuoteResponse, error) {
	quote, err := o.pricing.Calculate(ctx, req.FlightInstanceID, req.FareClassCode, req.Passengers)
	if err != nil {
		return nil, errors.InternalWrap("failed to calculate price", err)
	}

	quoteID := "qt_" + randomCode(12)
	q := PriceQuote{
		FlightInstanceID: req.FlightInstanceID,
		FareClassCode:    req.FareClassCode,
		Passengers:       req.Passengers,
		TotalAmount:      quote.TotalAmount,
		ExpiresAt:        quote.ExpiresAt,
	}
	data, _ := json.Marshal(q)
	if err := o.kv.Set(ctx, "quote:"+quoteID, string(data), quoteTTL); err != nil {
		return nil, errors.InternalWrap("failed to cache quote", err)
	}

	breakdown := o.pricing.GetBreakdown(quote.TotalAmount / float64(req.Passengers))

	return &PriceQuoteResponse{
		QuoteID:       quoteID,
		FlightID:      req.FlightInstanceID,
		FareClassCode: req.FareClassCode,
		Price:         breakdown.BaseFare,
		Taxes:         breakdown.Total - breakdown.BaseFare - breakdown.Fees,
		Total:         quote.TotalAmount,
		Breakdown:     breakdown,
		Currency:      "USD",
		ExpiresAt:     quote.ExpiresAt,
	}, nil
}

func (o *Orchestrator) CreateBooking(ctx context.Context, req CreateBookingRequest) (*CreateBookingResponse, error) {
	// Idempotency: replay a cached result if present.
	if req.IdempotencyKey != "" {
		if cached, err := o.kv.GetString(ctx, "idem:"+req.IdempotencyKey); err == nil && cached != "" {
			var resp CreateBookingResponse
			if err := json.Unmarshal([]byte(cached), &resp); err == nil {
				return &resp, nil
			}
		}
	}

	quote, err := o.loadQuote(ctx, req.QuoteID)
	if err != nil {
		return nil, err
	}
	if time.Now().After(quote.ExpiresAt) {
		_ = o.kv.Del(ctx, "quote:"+req.QuoteID)
		return nil, errors.ErrQuoteExpired
	}

	if len(req.Passengers) != quote.Passengers {
		return nil, errors.Validation("passenger count does not match the quote")
	}

	fareClassID, _, err := o.flights.ResolveFareClass(ctx, quote.FlightInstanceID, quote.FareClassCode)
	if err != nil {
		return nil, errors.InternalWrap("failed to resolve fare class", err)
	}
	if fareClassID == 0 {
		return nil, errors.Validation("fare class not found for this flight")
	}

	pnrCode := generatePNR()
	holdReq := inventory.HoldRequest{
		FlightInstanceID: quote.FlightInstanceID,
		FareClassID:      fareClassID,
		Passengers:       quote.Passengers,
		UserID:           userIDFromContext(ctx, o.logger),
		PNRCode:          pnrCode,
	}
	if err := o.inv.HoldSeats(ctx, holdReq); err != nil {
		return nil, err
	}

	pi, err := o.payments.CreatePaymentIntent(ctx, 0, quote.TotalAmount, "USD")
	if err != nil {
		_ = o.inv.ReleaseHolds(ctx, pnrCode)
		return nil, errors.ErrPaymentFailed
	}

	now := time.Now().UTC()
	expiresAt := now.Add(pnrTTL)
	pnr := &PNR{
		Code:        pnrCode,
		UserID:      holdReq.UserID,
		Status:      PNRPending,
		TotalAmount: quote.TotalAmount,
		Currency:    "USD",
		ExpiresAt:   &expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	pnrID, err := o.bookings.CreatePNR(ctx, pnr)
	if err != nil {
		_ = o.inv.ReleaseHolds(ctx, pnrCode)
		_, _ = o.payments.Refund(ctx, pi.ID, nil)
		return nil, errors.InternalWrap("failed to persist pnr", err)
	}
	pnr.ID = pnrID

	for range req.Passengers {
		b := &Booking{
			PNRID:            pnrID,
			FlightInstanceID: quote.FlightInstanceID,
			FareClassID:      fareClassID,
			Status:           BookingPending,
			BasePrice:        0,
			TotalPrice:       0,
			Currency:         "USD",
			CreatedAt:        now,
		}
		if _, err := o.bookings.CreateBooking(ctx, b); err != nil {
			_ = o.inv.ReleaseHolds(ctx, pnrCode)
			_, _ = o.payments.Refund(ctx, pi.ID, nil)
			return nil, errors.InternalWrap("failed to persist booking", err)
		}
	}

	resp := CreateBookingResponse{
		PNR:                 pnrCode,
		PNRID:               pnrID,
		Status:              string(PNRPending),
		PaymentIntentID:     pi.ID,
		PaymentClientSecret: pi.ClientSecret,
		AmountCents:         pi.Amount,
		Currency:            "USD",
		HoldExpiresAt:       now.Add(pnrTTL),
	}

	pnrData, _ := json.Marshal(resp)
	if err := o.kv.Set(ctx, "pnr:"+pnrCode, string(pnrData), pnrTTL); err != nil {
		o.logger.Warn("failed to cache pnr", zap.String("pnr", pnrCode), zap.Error(err))
	}
	if req.IdempotencyKey != "" {
		_ = o.kv.Set(ctx, "idem:"+req.IdempotencyKey, string(pnrData), idemTTL)
	}

	if err := o.notify.SendBookingConfirmation(ctx, pnrID); err != nil {
		o.logger.Warn("notification failed", zap.String("pnr", pnrCode), zap.Error(err))
	}

	o.logger.Info("booking created",
		zap.String("pnr", pnrCode),
		zap.Int64("pnr_id", pnrID),
		zap.Int64("flight_instance_id", quote.FlightInstanceID),
		zap.Float64("total", quote.TotalAmount))

	return &resp, nil
}

func (o *Orchestrator) ConfirmBooking(ctx context.Context, pnrCode string) (*ConfirmResponse, error) {
	if pnrCode == "" {
		return nil, errors.Validation("PNR is required")
	}
	pnr, err := o.bookings.GetPNRByCode(ctx, pnrCode)
	if err != nil {
		return nil, errors.InternalWrap("failed to load pnr", err)
	}
	if pnr == nil {
		return nil, errors.ErrPNRNotFound
	}
	if pnr.Status != PNRPending {
		return nil, errors.ErrBookingAlreadyConfirmed
	}

	paymentID := o.paymentIntentForPNR(ctx, pnrCode)
	status, vErr := o.payments.VerifyPayment(ctx, paymentID)
	if vErr != nil {
		return nil, errors.InternalWrap("failed to verify payment", vErr)
	}
	if status != "succeeded" {
		return nil, errors.ErrPaymentNotConfirmed
	}

	if err := o.bookings.UpdatePNRStatus(ctx, pnr.ID, PNRConfirmed); err != nil {
		return nil, errors.InternalWrap("failed to confirm pnr", err)
	}
	if err := o.inv.ConfirmHolds(ctx, pnrCode); err != nil {
		o.logger.Warn("failed to confirm holds", zap.String("pnr", pnrCode), zap.Error(err))
	}
	_ = o.kv.Del(ctx, "pnr:"+pnrCode)

	o.logger.Info("booking confirmed", zap.String("pnr", pnrCode), zap.Int64("pnr_id", pnr.ID))
	return &ConfirmResponse{PNR: pnrCode, Status: string(PNRConfirmed)}, nil
}

func (o *Orchestrator) CancelBooking(ctx context.Context, pnrCode string) (*CancelResponse, error) {
	if pnrCode == "" {
		return nil, errors.Validation("PNR is required")
	}
	pnr, err := o.bookings.GetPNRByCode(ctx, pnrCode)
	if err != nil {
		return nil, errors.InternalWrap("failed to load pnr", err)
	}
	if pnr == nil {
		return nil, errors.ErrPNRNotFound
	}
	if pnr.Status == PNRCancelled {
		return &CancelResponse{PNR: pnrCode, Status: string(PNRCancelled)}, nil
	}
	if pnr.Status == PNRConfirmed {
		// Attempt a refund; best-effort.
		o.maybeRefund(ctx, pnr)
	}

	if err := o.bookings.UpdatePNRStatus(ctx, pnr.ID, PNRCancelled); err != nil {
		return nil, errors.InternalWrap("failed to cancel pnr", err)
	}
	if err := o.inv.ReleaseHolds(ctx, pnrCode); err != nil {
		o.logger.Warn("failed to release holds", zap.String("pnr", pnrCode), zap.Error(err))
	}
	_ = o.kv.Del(ctx, "pnr:"+pnrCode)

	o.logger.Info("booking cancelled", zap.String("pnr", pnrCode))
	return &CancelResponse{PNR: pnrCode, Status: string(PNRCancelled)}, nil
}

func (o *Orchestrator) GetByPNR(ctx context.Context, pnrCode string) (*BookingDetail, error) {
	pnr, err := o.bookings.GetPNRByCode(ctx, pnrCode)
	if err != nil {
		return nil, errors.InternalWrap("failed to load pnr", err)
	}
	if pnr == nil {
		return nil, errors.ErrPNRNotFound
	}
	bookings, err := o.bookings.GetBookingsByPNR(ctx, pnr.ID)
	if err != nil {
		return nil, errors.InternalWrap("failed to load bookings", err)
	}
	var entries []BookingDetailEntry
	for _, b := range bookings {
		entries = append(entries, BookingDetailEntry{
			ID:     b.ID,
			Seat:   seatLabel(b.SeatID),
			Status: string(b.Status),
		})
	}
	return &BookingDetail{
		PNR:        pnrCode,
		Status:     string(pnr.Status),
		TotalCents: toCents(pnr.TotalAmount),
		Currency:   pnr.Currency,
		CreatedAt:  pnr.CreatedAt,
		Bookings:   entries,
	}, nil
}

func toCents(amount float64) int64 {
	return int64(amount*100 + 0.5)
}

func (o *Orchestrator) SelectSeats(ctx context.Context, pnrCode string, req SeatAssignmentRequest) (*SeatAssignmentResponse, error) {
	pnr, err := o.bookings.GetPNRByCode(ctx, pnrCode)
	if err != nil {
		return nil, errors.InternalWrap("failed to load pnr", err)
	}
	if pnr == nil {
		return nil, errors.ErrPNRNotFound
	}
	bookings, err := o.bookings.GetBookingsByPNR(ctx, pnr.ID)
	if err != nil {
		return nil, errors.InternalWrap("failed to load bookings", err)
	}
	if len(bookings) == 0 {
		return nil, errors.Validation("no bookings on this pnr")
	}
	seatID := parseSeatLabel(req.SeatNumber)
	if err := o.bookings.UpdateBookingSeat(ctx, bookings[0].ID, seatID); err != nil {
		return nil, errors.InternalWrap("failed to assign seat", err)
	}
	return &SeatAssignmentResponse{BookingID: bookings[0].ID, SeatID: seatID, Seat: req.SeatNumber}, nil
}

// loadQuote retrieves and unmarshals a cached quote.
func (o *Orchestrator) loadQuote(ctx context.Context, quoteID string) (*PriceQuote, error) {
	data, err := o.kv.GetString(ctx, "quote:"+quoteID)
	if err != nil {
		return nil, errors.ErrQuoteExpired
	}
	var q PriceQuote
	if err := json.Unmarshal([]byte(data), &q); err != nil {
		return nil, errors.ErrQuoteExpired
	}
	return &q, nil
}

// paymentIntentForPNR reads the payment intent id cached with the pnr.
func (o *Orchestrator) paymentIntentForPNR(ctx context.Context, pnrCode string) string {
	if v, err := o.kv.GetString(ctx, "pnr:"+pnrCode); err == nil {
		var cached CreateBookingResponse
		if err := json.Unmarshal([]byte(v), &cached); err == nil {
			return cached.PaymentIntentID
		}
	}
	return ""
}

func (o *Orchestrator) maybeRefund(ctx context.Context, pnr *PNR) {
	pid := o.paymentIntentForPNR(ctx, pnr.Code)
	if pid == "" {
		return
	}
	if _, err := o.payments.Refund(ctx, pid, &pnr.TotalAmount); err != nil {
		o.logger.Warn("refund failed", zap.Int64("pnr_id", pnr.ID), zap.Error(err))
	}
}

func seatLabel(id *int64) string {
	if id == nil {
		return ""
	}
	return seat.DecodeLabel(*id)
}

func userIDFromContext(ctx context.Context, log *zap.Logger) int64 {
	if id, ok := contextkeys.UserIDFromContext(ctx); ok {
		return id
	}
	log.Warn("user id missing from context; defaulting to 0")
	return 0
}

func parseSeatLabel(s string) *int64 {
	row, col, ok := seat.ParseLabel(s)
	if !ok {
		return nil
	}
	id := seat.EncodeID(row, col)
	return &id
}

func generatePNR() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = pnrCharset[randInt63()%int64(len(pnrCharset))]
	}
	return string(b)
}

func randomCode(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = pnrCharset[randInt63()%int64(len(pnrCharset))]
	}
	return string(b)
}

func randInt63() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	v := int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
		int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
	if v < 0 {
		v = -v
	}
	return v
}
