package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/api/helper"
	"github.com/you/airline-ticketing/internal/domain/checkin"
)

type CheckInHandler struct {
	svc *checkin.Service
	log *zap.Logger
}

func NewCheckInHandler(svc *checkin.Service, log *zap.Logger) *CheckInHandler {
	return &CheckInHandler{svc: svc, log: log}
}

func (h *CheckInHandler) Initiate(w http.ResponseWriter, r *http.Request) {
	pnr := chi.URLParam(r, "pnr")
	out, err := h.svc.InitiateCheckIn(r.Context(), pnr)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, out)
}

type assignSeatRequest struct {
	SeatNumber string `json:"seat_number" validate:"required"`
}

func (h *CheckInHandler) AssignSeat(w http.ResponseWriter, r *http.Request) {
	pnr := chi.URLParam(r, "pnr")
	var in assignSeatRequest
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	out, err := h.svc.AssignSeat(r.Context(), pnr, in.SeatNumber)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, out)
}

func (h *CheckInHandler) GetBoardingPass(w http.ResponseWriter, r *http.Request) {
	pnr := chi.URLParam(r, "pnr")
	out, err := h.svc.GetBoardingPass(r.Context(), pnr)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, out)
}
