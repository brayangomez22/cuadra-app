// Package receivables wires the receivables module: the credit of the
// customers, the receivables of the credit sales and the payments that
// settle them. internal/server builds it with New, giving it the customers
// and the cash drawer through the ports of app, and mounts its routes.
package receivables

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Customers and Cash are the customers module and the sales module's
	// cash sessions, adapted by internal/server.
	Customers      app.CustomerDirectory
	Cash           app.CashDrawer
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	// Now is the clock of the use cases; nil is time.Now (the demo
	// generator simulates one).
	Now func() time.Time
}

// Module is the wired receivables module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Logger == nil {
		return nil, errors.New("receivables: missing module dependency")
	}
	svc, err := app.NewService(app.Deps{
		Credit:         postgres.NewCreditRepository(cfg.DB),
		Customers:      cfg.Customers,
		Cash:           cfg.Cash,
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Now:            cfg.Now,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "receivables") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}
