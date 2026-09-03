package handler

import (
	"net/http"
	"strconv"

	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/you/airline-ticketing/internal/api/helper"
	"github.com/you/airline-ticketing/internal/domain/flight"
)

type FlightHandler struct {
	svc *flight.Service
	log *zap.Logger
}

func NewFlightHandler(svc *flight.Service, log *zap.Logger) *FlightHandler {
	return &FlightHandler{svc: svc, log: log}
}

func (h *FlightHandler) Search(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Search(r.Context(), flight.SearchCriteria{
		Origin:      flight.AirportCode(nonEmpty(r, "origin")),
		Destination: flight.AirportCode(nonEmpty(r, "destination")),
		Date:        parseDate(r, "date"),
		Passengers:  helper.QueryInt(r, "passengers", 1),
		CabinClass:  r.URL.Query().Get("cabin"),
	})
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, searchEnvelope{Data: d, Meta: searchMeta{Total: len(d), Currency: "USD"}})
}

func (h *FlightHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	id := parseID(r, "id")
	if id <= 0 {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flight id"})
		return
	}
	d, err := h.svc.GetDetail(r.Context(), id)
	if err != nil {
		helper.Error(w, err)
		return
	}
	if d == nil {
		helper.JSON(w, http.StatusNotFound, map[string]string{"error": "flight not found"})
		return
	}
	helper.JSON(w, http.StatusOK, d)
}

func (h *FlightHandler) GetSeatMap(w http.ResponseWriter, r *http.Request) {
	id := parseID(r, "id")
	if id <= 0 {
		helper.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid flight id"})
		return
	}
	sm, err := h.svc.GetSeatMap(r.Context(), id)
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, sm)
}

func (h *FlightHandler) ListAirports(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAirports(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: list})
}

func (h *FlightHandler) ListAirlines(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAirlines(r.Context())
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: list})
}

func (h *FlightHandler) ListAircraft(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAircraft(r.Context())
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: list})
}

func (h *FlightHandler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListRoutes(r.Context())
	if err != nil {
		helper.Error(w, err)
		return
	}
	helper.JSON(w, http.StatusOK, listEnvelope{Data: list})
}

func (h *FlightHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
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

func (h *FlightHandler) CreateFlightSchedule(w http.ResponseWriter, r *http.Request) {
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

func (h *FlightHandler) Materialize(w http.ResponseWriter, r *http.Request) {
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

func (h *FlightHandler) UpdateFlightInstance(w http.ResponseWriter, r *http.Request) {
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

func (h *FlightHandler) DeleteFlightInstance(w http.ResponseWriter, r *http.Request) {
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

type searchEnvelope struct {
	Data []flight.SearchResult `json:"data"`
	Meta searchMeta            `json:"meta"`
}

type searchMeta struct {
	Total    int    `json:"total"`
	Currency string `json:"currency"`
}

func parseID(r *http.Request, key string) int64 {
	id, err := strconv.ParseInt(chi.URLParam(r, key), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func nonEmpty(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func parseDate(r *http.Request, key string) time.Time {
	return parseDateLoose(r.URL.Query().Get(key))
}
