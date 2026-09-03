package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/you/airline-ticketing/internal/api/handler"
	"github.com/you/airline-ticketing/internal/api/middleware"
	"github.com/you/airline-ticketing/internal/pkg/config"
	"github.com/you/airline-ticketing/internal/pkg/store"
)

type HandlerGroup struct {
	Flight       *handler.FlightHandler
	User         *handler.UserHandler
	Booking      *handler.BookingHandler
	Payment      *handler.PaymentHandler
	Checkin      *handler.CheckInHandler
	Notification *handler.NotificationHandler

	AdminFlight   *handler.AdminFlightHandler
	AdminBooking  *handler.AdminBookingHandler
	AdminUser     *handler.AdminUserHandler
	AdminAircraft *handler.AdminAircraftHandler
	AdminRoute    *handler.AdminRouteHandler
	AdminPricing  *handler.AdminPricingHandler
	AdminReport   *handler.AdminReportHandler
}

func NewRouter(cfg *config.Config, kv store.KV, log *zap.Logger, h *HandlerGroup) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.Recoverer(log))
	r.Use(chiMiddleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "Stripe-Signature"},
		ExposedHeaders:   []string{"X-Request-ID", "X-Idempotency-Replay"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	limiter := middleware.NewIPLimiter(rate.Limit(cfg.RateLimitPerSec), cfg.RateLimitPerSec)
	r.Use(middleware.RateLimit(limiter))

	auth := middleware.JWTAuth(cfg.JWTSecret, cfg.JWTIssuer, log)

	r.Group(func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		})

		r.Get("/flights/search", h.Flight.Search)
		r.Get("/flights/{id}", h.Flight.GetDetail)
		r.Get("/flights/{id}/seat-map", h.Flight.GetSeatMap)
		r.Get("/airports", h.Flight.ListAirports)
		r.Get("/airlines", h.Flight.ListAirlines)
		r.Get("/aircraft", h.Flight.ListAircraft)
		r.Get("/routes", h.Flight.ListRoutes)

		r.Post("/auth/register", h.User.Register)
		r.Post("/auth/login", h.User.Login)
		r.Post("/auth/refresh", h.User.Refresh)

		r.Post("/bookings/price", h.Booking.GetPrice)
		r.Post("/bookings", h.Booking.Create)
		r.Post("/bookings/{pnr}/confirm", h.Booking.Confirm)
		r.Get("/bookings/{pnr}", h.Booking.GetByPNR)

		r.Post("/payments/create-intent", h.Payment.CreateIntent)
		r.Post("/payments/webhook", h.Payment.StripeWebhook)
		r.Post("/payments/refund", h.Payment.Refund)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth)

		r.Get("/users/me", h.User.GetProfile)
		r.Put("/users/me", h.User.UpdateProfile)

		r.Put("/bookings/{pnr}/seats", h.Booking.SelectSeats)
		r.Delete("/bookings/{pnr}", h.Booking.Cancel)

		r.Post("/check-in/{pnr}", h.Checkin.Initiate)
		r.Put("/check-in/{pnr}/assign-seat", h.Checkin.AssignSeat)
		r.Get("/check-in/{pnr}/boarding-pass", h.Checkin.GetBoardingPass)

		r.Post("/notifications/send", h.Notification.Send)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth)
		r.Use(middleware.RequireRole("ADMIN"))

		r.Get("/admin/flights", h.AdminFlight.List)
		r.Post("/admin/flights", h.AdminFlight.Create)
		r.Put("/admin/flights/{id}", h.AdminFlight.Update)
		r.Delete("/admin/flights/{id}", h.AdminFlight.Delete)
		r.Post("/admin/flights/materialize", h.AdminFlight.Materialize)

		r.Get("/admin/bookings", h.AdminBooking.List)
		r.Get("/admin/bookings/{id}", h.AdminBooking.Get)
		r.Put("/admin/bookings/{id}/status", h.AdminBooking.UpdateStatus)

		r.Get("/admin/users", h.AdminUser.List)
		r.Put("/admin/users/{id}", h.AdminUser.Update)

		r.Get("/admin/aircraft", h.AdminAircraft.List)
		r.Get("/admin/routes", h.AdminRoute.List)
		r.Post("/admin/routes", h.AdminRoute.Create)

		r.Get("/admin/pricing-rules", h.AdminPricing.List)
		r.Post("/admin/pricing-rules", h.AdminPricing.Create)

		r.Get("/admin/reports/load-factor", h.AdminReport.LoadFactor)
		r.Get("/admin/reports/revenue", h.AdminReport.Revenue)
		r.Get("/admin/reports/peak-times", h.AdminReport.PeakTimes)
	})

	return r
}
