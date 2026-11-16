// Package audit wires the audit module: the tenant's audit log, which the
// other modules write through a port inside the transaction of each
// audited operation, and its query for the owner. internal/server builds
// it with New and mounts its routes.
package audit

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB             *db.DB
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	// Now is the clock of the events; nil is time.Now (the demo generator
	// simulates one).
	Now func() time.Time
}

// Module is the wired audit module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Logger == nil {
		return nil, errors.New("audit: missing module dependency")
	}
	svc, err := app.NewService(app.Deps{
		Events:         postgres.NewEventRepository(cfg.DB),
		Now:            cfg.Now,
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "audit") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}
