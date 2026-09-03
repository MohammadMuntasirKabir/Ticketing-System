package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/you/airline-ticketing/internal/api"
	"github.com/you/airline-ticketing/internal/api/handler"
	"github.com/you/airline-ticketing/internal/domain/booking"
	"github.com/you/airline-ticketing/internal/domain/checkin"
	"github.com/you/airline-ticketing/internal/domain/flight"
	"github.com/you/airline-ticketing/internal/domain/inventory"
	"github.com/you/airline-ticketing/internal/domain/notification"
	"github.com/you/airline-ticketing/internal/domain/payment"
	"github.com/you/airline-ticketing/internal/domain/pricing"
	"github.com/you/airline-ticketing/internal/domain/report"
	"github.com/you/airline-ticketing/internal/domain/user"
	"github.com/you/airline-ticketing/internal/pkg/config"
	"github.com/you/airline-ticketing/internal/pkg/logger"
	"github.com/you/airline-ticketing/internal/pkg/store"
	"github.com/you/airline-ticketing/internal/repository/bookingrepo"
	"github.com/you/airline-ticketing/internal/repository/flightrepo"
	"github.com/you/airline-ticketing/internal/repository/userrepo"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		panic("invalid configuration: " + err.Error())
	}

	log, err := logger.New(cfg.LogLevel, string(cfg.Environment))
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer log.Sync()

	// The KV store backs quotes, inventory holds, idempotency keys and
	// refresh tokens. Prefer Redis when reachable, otherwise fall back to
	// an in-process store so the API runs without external services.
	kv := newKV(cfg, log)

	ctx := context.Background()

	var flightRepo flight.Repository
	var userRepo user.Repository
	var bookingRepo booking.Repository

	if cfg.DatabaseURL != "" && !strings.Contains(cfg.DatabaseURL, "mode=memory") {
		db, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			panic("failed to connect to database: " + err.Error())
		}
		defer db.Close()
		if err := db.PingContext(ctx); err != nil {
			log.Warn("database unavailable; falling back to in-memory", zap.Error(err))
			flightRepo = flightrepo.New()
			userRepo = userrepo.New()
			bookingRepo = bookingrepo.New()
		} else {
			flightRepo = flightrepo.NewPostgres(db)
			userRepo = userrepo.NewPostgres(db)
			bookingRepo = bookingrepo.NewPostgres(db)
			log.Info("connected to PostgreSQL", zap.String("url", cfg.DatabaseURL))
		}
	} else {
		flightRepo = flightrepo.New()
		userRepo = userrepo.New()
		bookingRepo = bookingrepo.New()
	}

	seedAdmin(ctx, userRepo, cfg, log)

	flightSvc := flight.NewService(flightRepo, log)
	pricingSvc := pricing.NewEngine(log)
	inventorySvc := inventory.NewService(kv, flightRepo, log)
	userSvc := user.NewService(userRepo, store.NewKVTokenStore(kv), nil, cfg, log)
	notifSvc := notification.NewService(log)
	checkinSvc := checkin.NewService(bookingRepo, flightRepo, log)
	reportSvc := report.NewService(flightRepo, bookingRepo, log)

	paymentProvider, webhook := newPayment(cfg, log)

	orchestrator := booking.NewOrchestrator(
		flightRepo,
		inventorySvc,
		pricingSvc,
		paymentProvider,
		bookingRepo,
		kv,
		notifSvc,
		log,
	)

	h := &api.HandlerGroup{
		Flight:        handler.NewFlightHandler(flightSvc, log),
		User:          handler.NewUserHandler(userSvc, log),
		Booking:       handler.NewBookingHandler(orchestrator, log),
		Payment:       handler.NewPaymentHandler(paymentProvider, webhook, log),
		Checkin:       handler.NewCheckInHandler(checkinSvc, log),
		Notification:  handler.NewNotificationHandler(notifSvc, log),
		AdminFlight:   handler.NewAdminFlightHandler(flightSvc, log),
		AdminBooking:  handler.NewAdminBookingHandler(bookingRepo, log),
		AdminUser:     handler.NewAdminUserHandler(userSvc, log),
		AdminAircraft: handler.NewAdminAircraftHandler(flightSvc, log),
		AdminRoute:    handler.NewAdminRouteHandler(flightSvc, log),
		AdminPricing:  handler.NewAdminPricingHandler(pricingSvc, log),
		AdminReport:   handler.NewAdminReportHandler(reportSvc, log),
	}

	r := api.NewRouter(cfg, kv, log, h)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	kv.Close()
}

// newKV returns a Redis-backed store when configured and reachable,
// falling back to an in-memory store so local runs need no broker.
func newKV(cfg *config.Config, log *zap.Logger) store.KV {
	if cfg.RedisAddr != "" {
		r := store.NewRedis(cfg.RedisAddr, cfg.RedisPassword)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := r.Client().Ping(ctx).Err()
		cancel()
		if err == nil {
			return r
		}
		log.Warn("redis unavailable; using in-memory store", zap.String("addr", cfg.RedisAddr), zap.Error(err))
	}
	return store.NewInMemory()
}

// newPayment returns a Stripe-backed provider when a secret is
// configured, otherwise an in-memory fake so the booking flow works
// locally without payment credentials.
func newPayment(cfg *config.Config, log *zap.Logger) (payment.Provider, payment.WebhookVerifier) {
	if cfg.StripeSecretKey == "" {
		return payment.NewFake(), nil
	}
	provider := payment.NewStripeProvider(cfg.StripeSecretKey, cfg.StripeWebhookSec, log)
	var webhook payment.WebhookVerifier
	if cfg.StripeWebhookSec != "" {
		webhook = provider
	}
	return provider, webhook
}

// seedAdmin provisions a default ADMIN account so the admin endpoints are
// usable out of the box in development.
func seedAdmin(ctx context.Context, repo user.Repository, cfg *config.Config, log *zap.Logger) {
	email := getEnv("AL_ADMIN_EMAIL", "admin@airline.local")
	password := getEnv("AL_ADMIN_PASSWORD", "admin12345")

	if existing, err := repo.GetByEmail(ctx, email); err == nil && existing != nil {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash admin password", zap.Error(err))
		return
	}
	now := time.Now().UTC()
	_, err = repo.Create(ctx, &user.User{
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    "Admin",
		LastName:     "User",
		Role:         user.RoleAdmin,
		Status:       "ACTIVE",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		log.Error("failed to seed admin user", zap.Error(err))
		return
	}
	log.Info("seeded admin user", zap.String("email", email))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
