// Package config loads the application configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Env             string
	HTTPAddr        string
	DatabaseURL     string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
}

// Load reads the configuration using getenv (os.Getenv in production). It
// reports every missing or invalid variable at once so startup fails fast.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Env:             withDefault(getenv("APP_ENV"), "development"),
		HTTPAddr:        withDefault(getenv("HTTP_ADDR"), ":8080"),
		DatabaseURL:     getenv("DATABASE_URL"),
		ShutdownTimeout: 15 * time.Second,
	}
	var errs []error

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(withDefault(getenv("LOG_LEVEL"), "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL is invalid: %w", err))
	}

	if raw := getenv("SHUTDOWN_TIMEOUT"); raw != "" {
		d, err := time.ParseDuration(raw)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT is invalid: %w", err))
		case d <= 0:
			errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT must be positive, got %s", d))
		default:
			cfg.ShutdownTimeout = d
		}
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func withDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
