package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/api/helper"
	"github.com/you/airline-ticketing/internal/domain/booking"
)

type BookingHandler struct {
	orch *booking.Orchestrator
	log  *zap.Logger
}

func NewBookingHandler(orch *booking.Orchestrator, log *zap.Logger) *BookingHandler {
	return &BookingHandler{orch: orch, log: log}
}

func (h *BookingHandler) GetPrice(w http.ResponseWriter, r *http.Request) {
	var in booking.PriceQuoteRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	out, err := h.orch.GetPriceQuote(r.Context(), in)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, out)
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in booking.CreateBookingRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	out, err := h.orch.CreateBooking(r.Context(), in)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusCreated, out)
}

func (h *BookingHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	pnr := chi.URLParam(r, "pnr")
	out, err := h.orch.ConfirmBooking(r.Context(), pnr)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, out)
}

func (h *BookingHandler) GetByPNR(w http.ResponseWriter, r *http.Request) {
	pnr := chi.URLParam(r, "pnr")
	out, err := h.orch.GetByPNR(r.Context(), pnr)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, out)
}

func (h *BookingHandler) SelectSeats(w http.ResponseWriter, r *http.Request) {
	pnr := chi.URLParam(r, "pnr")
	var in booking.SeatAssignmentRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	out, err := h.orch.SelectSeats(r.Context(), pnr, in)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, out)
}

func (h *BookingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	pnr := chi.URLParam(r, "pnr")
	out, err := h.orch.CancelBooking(r.Context(), pnr)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, out)
}
