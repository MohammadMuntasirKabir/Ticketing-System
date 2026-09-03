package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

// Environment tags runtime behavior (logging format, cookie security).
type Environment string

const (
	EnvDev  Environment = "dev"
	EnvProd Environment = "prod"
)

// Config holds all runtime configuration loaded from the environment.
// Secrets are never defaulted in production; Load() returns safe dev
// defaults only for local development convenience.
type Config struct {
	Environment        Environment
	Port               string
	DatabaseURL        string
	RedisAddr          string
	RedisPassword      string
	JWTSecret          string
	JWTExpiryMinutes   int
	RefreshExpiryDays  int
	StripeSecretKey    string
	StripeWebhookSec   string
	AllowedOrigins     []string
	RateLimitPerSec    int
	JWTIssuer          string
	AccessTokenCookie  string
	RefreshTokenCookie string
	LogLevel           zapcore.Level
}

func Load() *Config {
	return &Config{
		Environment:        envEnum("AL_ENV", "dev"),
		Port:               getEnv("AL_PORT", "8080"),
		DatabaseURL:        getEnv("AL_DATABASE_URL", "postgres://airline:airline_dev@localhost:5432/airline?sslmode=disable"),
		RedisAddr:          getEnv("AL_REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("AL_REDIS_PASSWORD", ""),
		JWTSecret:          getEnv("AL_JWT_SECRET", "dev-jwt-secret-change-in-prod"),
		JWTExpiryMinutes:   getEnvInt("AL_JWT_EXPIRY", 30),
		RefreshExpiryDays:  getEnvInt("AL_REFRESH_EXPIRY", 7),
		StripeSecretKey:    getEnv("AL_STRIPE_SECRET_KEY", ""),
		StripeWebhookSec:   getEnv("AL_STRIPE_WEBHOOK_SECRET", ""),
		AllowedOrigins:     getEnvStrings("AL_ALLOWED_ORIGINS", "http://localhost:3000"),
		RateLimitPerSec:    getEnvInt("AL_RATE_LIMIT_PER_SEC", 20),
		JWTIssuer:          getEnv("AL_JWT_ISSUER", "airline-ticketing"),
		AccessTokenCookie:  getEnv("AL_ACCESS_TOKEN_COOKIE", "al_access_token"),
		RefreshTokenCookie: getEnv("AL_REFRESH_TOKEN_COOKIE", "al_refresh_token"),
		LogLevel:           parseLogLevel(getEnv("AL_LOG_LEVEL", "info")),
	}
}

// Validate ensures configuration is safe to run with. It is intentionally
// strict in production so misconfigured deployments fail loudly.
func (c *Config) Validate() error {
	if c.Environment == EnvProd {
		if c.JWTSecret == "" || strings.Contains(c.JWTSecret, "change") {
			return errors.New("AL_JWT_SECRET must be set to a strong value in production")
		}
		if c.StripeSecretKey == "" {
			return errors.New("AL_STRIPE_SECRET_KEY must be set in production")
		}
		if c.StripeWebhookSec == "" {
			return errors.New("AL_STRIPE_WEBHOOK_SECRET must be set in production")
		}
	}
	if c.RateLimitPerSec <= 0 {
		c.RateLimitPerSec = 20
	}
	if c.JWTExpiryMinutes <= 0 {
		c.JWTExpiryMinutes = 30
	}
	if c.RefreshExpiryDays <= 0 {
		c.RefreshExpiryDays = 7
	}
	if len(c.AllowedOrigins) == 0 {
		c.AllowedOrigins = []string{"http://localhost:3000"}
	}
	return nil
}

func (c *Config) RefreshTTL() time.Duration {
	return time.Duration(c.RefreshExpiryDays) * 24 * time.Hour
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvStrings(key, fallback string) []string {
	v := getEnv(key, fallback)
	parts := strings.Split(v, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseLogLevel(s string) zapcore.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func envEnum(key string, fallback Environment) Environment {
	v := strings.ToLower(getEnv(key, string(fallback)))
	switch v {
	case "prod":
		return EnvProd
	case "dev":
		return EnvDev
	default:
		return EnvDev
	}
}
