// Package config loads the application configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// minJWTSecretLength is the minimum HS256 key size (RFC 7518 §3.2).
const minJWTSecretLength = 32

// Config holds the application configuration.
type Config struct {
	Env         string
	HTTPAddr    string
	DatabaseURL string
	// JWTSecret signs the access tokens (HS256). Never logged.
	JWTSecret       []byte
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	// SentryDSN enables error reporting to Sentry; empty disables it. Never logged.
	SentryDSN string
	// SentryRelease names the build in Sentry; empty means the VCS revision.
	SentryRelease string
	// DemoEnabled resets the demo tenants every night (T34) and lets anyone
	// enter them without a password (POST /auth/demo, T34a); DemoPassword is
	// the password of their users, public on purpose. Never logged.
	DemoEnabled  bool
	DemoPassword string
}

// minDemoPassword is the shortest demo password accepted.
const minDemoPassword = 12

// Load reads the configuration using getenv (os.Getenv in production). It
// reports every missing or invalid variable at once so startup fails fast.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Env:             withDefault(getenv("APP_ENV"), "development"),
		HTTPAddr:        withDefault(getenv("HTTP_ADDR"), ":8080"),
		DatabaseURL:     getenv("DATABASE_URL"),
		ShutdownTimeout: 15 * time.Second,
		SentryDSN:       getenv("SENTRY_DSN"),
		SentryRelease:   getenv("SENTRY_RELEASE"),
	}
	var errs []error

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}

	secret, err := jwtSecret(getenv("JWT_SECRET"))
	if err != nil {
		errs = append(errs, err)
	}
	cfg.JWTSecret = secret

	if err := cfg.LogLevel.UnmarshalText([]byte(withDefault(getenv("LOG_LEVEL"), "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL is invalid: %w", err))
	}

	if raw := getenv("SHUTDOWN_TIMEOUT"); raw != "" {
		d, err := shutdownTimeout(raw)
		if err != nil {
			errs = append(errs, err)
		} else {
			cfg.ShutdownTimeout = d
		}
	}

	if err := cfg.loadDemo(getenv); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return cfg, nil
}

// loadDemo reads DEMO_ENABLED and, when it is on, DEMO_PASSWORD.
func (cfg *Config) loadDemo(getenv func(string) string) error {
	if raw := getenv("DEMO_ENABLED"); raw != "" {
		on, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("DEMO_ENABLED is invalid: %w", err)
		}
		cfg.DemoEnabled = on
	}
	cfg.DemoPassword = getenv("DEMO_PASSWORD")
	if cfg.DemoEnabled && len(cfg.DemoPassword) < minDemoPassword {
		return fmt.Errorf("DEMO_PASSWORD must have at least %d characters when DEMO_ENABLED is on", minDemoPassword)
	}
	return nil
}

func withDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// jwtSecret checks the JWT secret. The value is never echoed: it is a
// secret.
func jwtSecret(secret string) ([]byte, error) {
	switch {
	case secret == "":
		return nil, errors.New("JWT_SECRET is required")
	case len(secret) < minJWTSecretLength:
		return nil, fmt.Errorf("JWT_SECRET must have at least %d bytes", minJWTSecretLength)
	}
	return []byte(secret), nil
}

// shutdownTimeout reads a positive duration.
func shutdownTimeout(raw string) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	switch {
	case err != nil:
		return 0, fmt.Errorf("SHUTDOWN_TIMEOUT is invalid: %w", err)
	case d <= 0:
		return 0, fmt.Errorf("SHUTDOWN_TIMEOUT must be positive, got %s", d)
	}
	return d, nil
}
