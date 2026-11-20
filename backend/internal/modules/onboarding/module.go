// Package onboarding wires the getting-started module: the checklist a new
// tenant follows (business data, invoicing, products, stock, team, first
// sale...), computed from its real data and with the steps of its business
// template. It reads the other modules' tables on PostgreSQL, and what has
// rules of another module (the template's steps, the active modules, the
// invoicing readiness) through ports. It owns only who hid the checklist.
// internal/server builds it with New and mounts its routes.
package onboarding

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Authorize is the permission matrix: each user sees the steps their
	// role can resolve.
	Authorize auth.Authorizer
	Templates app.Templates
	Modules   app.Modules
	Invoicing app.Invoicing
	Logger    *slog.Logger
	// Now decides which invitations are in force and stamps a dismissal;
	// nil is time.Now.
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Module is the wired onboarding module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Logger == nil || cfg.Authorize == nil {
		return nil, errors.New("onboarding: missing module dependency")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	svc, err := app.NewService(app.Deps{
		State:          postgres.NewStateRepository(cfg.DB, cfg.Now),
		Dismissals:     postgres.NewDismissalRepository(cfg.DB),
		Templates:      cfg.Templates,
		Modules:        cfg.Modules,
		Invoicing:      cfg.Invoicing,
		Can:            cfg.Authorize,
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

// RegisterRoutes mounts the module's operations (tag "onboarding") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}
