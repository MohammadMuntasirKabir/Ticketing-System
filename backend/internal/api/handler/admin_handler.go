package handler

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/api/helper"
	"github.com/you/airline-ticketing/internal/domain/booking"
	"github.com/you/airline-ticketing/internal/domain/flight"
	"github.com/you/airline-ticketing/internal/domain/pricing"
	"github.com/you/airline-ticketing/internal/domain/report"
	"github.com/you/airline-ticketing/internal/domain/user"
)

// ---- AdminFlight ----

type AdminFlightHandler struct {
	svc *flight.Service
	log *zap.Logger
}

func NewAdminFlightHandler(svc *flight.Service, log *zap.Logger) *AdminFlightHandler {
	return &AdminFlightHandler{svc: svc, log: log}
}

func (h *AdminFlightHandler) List(w http.ResponseWriter, r *http.Request) {
	instances, err := h.svc.ListFlightInstances(r.Context(), flight.ListQuery{
		Limit:  queryInt(r, "limit", 100),
		Offset: queryInt(r, "offset", 0),
	})
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: instances, Meta: listMeta{Total: len(instances)}})
}

func (h *AdminFlightHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in flight.ScheduleInput
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	id, err := h.svc.CreateFlightSchedule(r.Context(), in)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *AdminFlightHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := parseID(r, "id")
	if id <= 0 {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flight id"})
		return
	}
	var in flight.FlightInstanceUpdate
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	if err := h.svc.UpdateFlightInstance(r.Context(), id, in); err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *AdminFlightHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r, "id")
	if id <= 0 {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flight id"})
		return
	}
	if err := h.svc.DeleteFlightInstance(r.Context(), id); err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminFlightHandler) Materialize(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ScheduleID int64  `json:"schedule_id" validate:"required,gt=0"`
		Date       string `json:"date" validate:"required"`
	}
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	date := parseDateLoose(in.Date)
	if date.IsZero() {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date"})
		return
	}
	n, err := h.svc.MaterializeSchedule(r.Context(), in.ScheduleID, date)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, map[string]int{"created": n})
}

// ---- AdminBooking ----

type AdminBookingHandler struct {
	repo booking.Repository
	log  *zap.Logger
}

func NewAdminBookingHandler(repo booking.Repository, log *zap.Logger) *AdminBookingHandler {
	return &AdminBookingHandler{repo: repo, log: log}
}

func (h *AdminBookingHandler) List(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.repo.ListBookings(r.Context(), queryInt(r, "limit", 100), queryInt(r, "offset", 0))
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: bookings, Meta: listMeta{Total: len(bookings)}})
}

type adminBookingDetail struct {
	Booking    booking.Booking     `json:"booking"`
	Passengers []booking.Passenger `json:"passengers"`
}

func (h *AdminBookingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := parseID(r, "id")
	if id <= 0 {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid booking id"})
		return
	}
	b, err := h.repo.GetBooking(r.Context(), id)
	if err != nil {
		helper.Error(w, err)
		return
	}
	if b == nil {
		helper.JSON(w, http.StatusNotFound, map[string]string{"error": "booking not found"})
		return
	}
	passengers, err := h.repo.GetPassengersByBooking(r.Context(), id)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, adminBookingDetail{Booking: *b, Passengers: passengers})
}

func (h *AdminBookingHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := parseID(r, "id")
	if id <= 0 {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid booking id"})
		return
	}
	var in struct {
		Status booking.BookingStatus `json:"status" validate:"required"`
	}
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	if err := h.repo.UpdateBookingStatus(r.Context(), id, in.Status); err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ---- AdminUser ----

type AdminUserHandler struct {
	svc *user.Service
	log *zap.Logger
}

func NewAdminUserHandler(svc *user.Service, log *zap.Logger) *AdminUserHandler {
	return &AdminUserHandler{svc: svc, log: log}
}

func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, total, err := h.svc.List(r.Context(), r.URL.Query().Get("q"), queryInt(r, "limit", 100), queryInt(r, "offset", 0))
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: users, Meta: listMeta{Total: total}})
}

func (h *AdminUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := parseID(r, "id")
	if id <= 0 {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	var in struct {
		Role user.Role `json:"role" validate:"required"`
	}
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	if err := h.svc.UpdateRole(r.Context(), id, in.Role); err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ---- AdminAircraft ----

type AdminAircraftHandler struct {
	svc *flight.Service
	log *zap.Logger
}

func NewAdminAircraftHandler(svc *flight.Service, log *zap.Logger) *AdminAircraftHandler {
	return &AdminAircraftHandler{svc: svc, log: log}
}

func (h *AdminAircraftHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAircraft(r.Context())
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: list, Meta: listMeta{Total: len(list)}})
}

// ---- AdminRoute ----

type AdminRouteHandler struct {
	svc *flight.Service
	log *zap.Logger
}

func NewAdminRouteHandler(svc *flight.Service, log *zap.Logger) *AdminRouteHandler {
	return &AdminRouteHandler{svc: svc, log: log}
}

func (h *AdminRouteHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListRoutes(r.Context())
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: list, Meta: listMeta{Total: len(list)}})
}

func (h *AdminRouteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in flight.RouteInput
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	id, err := h.svc.CreateRoute(r.Context(), in)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// ---- AdminPricing ----

type AdminPricingHandler struct {
	engine *pricing.Engine
	log    *zap.Logger
}

func NewAdminPricingHandler(engine *pricing.Engine, log *zap.Logger) *AdminPricingHandler {
	return &AdminPricingHandler{engine: engine, log: log}
}

func (h *AdminPricingHandler) List(w http.ResponseWriter, r *http.Request) {
	helper.JSON(w, http.StatusOK, listEnvelope{Data: h.engine.ListRules(), Meta: listMeta{Total: len(h.engine.ListRules())}})
}

func (h *AdminPricingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FareClassCode string  `json:"fare_class_code" validate:"required"`
		BasePrice     float64 `json:"base_price" validate:"gte=0"`
	}
	if err := helper.Decode(r, &in); err != nil {
		helper.Error(w, err)
		return
	}
	rule, err := h.engine.CreateRule(in.FareClassCode, in.BasePrice)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusCreated, rule)
}

// ---- AdminReport ----

type AdminReportHandler struct {
	svc *report.Service
	log *zap.Logger
}

func NewAdminReportHandler(svc *report.Service, log *zap.Logger) *AdminReportHandler {
	return &AdminReportHandler{svc: svc, log: log}
}

func (h *AdminReportHandler) LoadFactor(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.LoadFactor(r.Context())
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: rows, Meta: listMeta{Total: len(rows)}})
}

func (h *AdminReportHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.Revenue(r.Context())
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: rows, Meta: listMeta{Total: len(rows)}})
}

func (h *AdminReportHandler) PeakTimes(w http.ResponseWriter, r *http.Request) {
	rows, err := h.svc.PeakTimes(r.Context())
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: rows, Meta: listMeta{Total: len(rows)}})
}
