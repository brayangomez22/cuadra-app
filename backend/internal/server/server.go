// Package server composes the API: it builds the modules on their shared
// infrastructure, joins them through ports and mounts them behind the
// contract's gates. cmd/api runs it, and the API tests (backend/test/api)
// build the same one.
package server

import (
	"log/slog"
	"net/http"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// Config are the dependencies of the API.
type Config struct {
	DB *db.DB
	// Tokens issues and verifies the access tokens.
	Tokens *auth.JWT
	// Jobs enqueues background jobs; the modules add their workers to
	// Workers, the bundle Jobs works (before it starts).
	Jobs    *jobs.Client
	Workers *river.Workers
	// Docs serves /docs and /openapi.json (development only).
	Docs           bool
	Log            *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// New builds every module and the handler that serves them. It fails, so
// the API does not start, if a module, a business template or the contract
// is invalid.
func New(cfg Config) (http.Handler, error) {
	w, err := newWiring(cfg)
	if err != nil {
		return nil, err
	}
	routes, err := w.modules()
	if err != nil {
		return nil, err
	}
	return newHandler(handlerConfig{
		Log:            cfg.Log,
		TracerProvider: cfg.TracerProvider,
		DB:             cfg.DB,
		Docs:           cfg.Docs,
		Verifier:       cfg.Tokens,
		Permissions:    w.permissions,
		Modules:        w.moduleService,
		Routes:         routes,
	})
}
