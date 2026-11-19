// Package reports wires the reports module: the business reports for the
// owner and the admin (sales, products, profit, idle stock, cash closings
// and the team's control) and their export to Excel or CSV. It only reads:
// the tables of the other modules on PostgreSQL, and the receivables'
// aging through a port. internal/server builds it with New and mounts its
// routes.
package reports

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Authorize decides who sees costs and margins.
	Authorize auth.Authorizer
	// Receivables is the receivables module's aging.
	Receivables    app.Receivables
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	// Now is the clock of "today"; nil is time.Now (the demo generator
	// simulates one).
	Now func() time.Time
}

// Module is the wired reports module.
type Module struct {
	service *app.Service
	can     auth.Authorizer
	log     *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Logger == nil || cfg.Authorize == nil {
		return nil, errors.New("reports: missing module dependency")
	}
	svc, err := app.NewService(app.Deps{
		Sales:          postgres.NewSalesRepository(cfg.DB),
		Stock:          postgres.NewStockRepository(cfg.DB),
		Team:           postgres.NewTeamRepository(cfg.DB),
		Receivables:    cfg.Receivables,
		Now:            cfg.Now,
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, can: cfg.Authorize, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "reports") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.can, m.log)
}
