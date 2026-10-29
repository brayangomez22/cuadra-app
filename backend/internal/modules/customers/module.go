// Package customers wires the customers module: the customers of a tenant
// and their segments. internal/server builds it with New and mounts its
// routes.
package customers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB             *db.DB
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	// Now is the clock of the use cases; nil is time.Now (the demo
	// generator simulates one).
	Now func() time.Time
}

// Module is the wired customers module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Logger == nil {
		return nil, errors.New("customers: missing module dependency")
	}
	svc, err := app.NewService(app.Deps{
		Customers:      postgres.NewCustomerRepository(cfg.DB),
		Segments:       postgres.NewSegmentRepository(cfg.DB),
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

// RegisterRoutes mounts the module's operations (tag "customers") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}

// SeedSegments gives a new tenant the customer segments of its business
// template. It is for the sign-up, through the port internal/server adapts;
// with the sign-up's ctx it joins its transaction.
func (m *Module) SeedSegments(ctx context.Context, tenantID uuid.UUID, names []string) error {
	return m.service.SeedSegments(ctx, tenantID, names)
}

// CheckCustomer tells whether a sale can be made to customerID (see
// app.Service.CheckCustomer). It is for the sales module, through the port
// internal/server adapts.
func (m *Module) CheckCustomer(ctx context.Context, tenantID, userID, customerID uuid.UUID) error {
	return m.service.CheckCustomer(ctx, app.Actor{TenantID: tenantID, UserID: userID}, customerID)
}

// CustomerExists tells whether customerID is a customer of tenantID, active
// or not (domain.ErrCustomerNotFound otherwise). It is for the receivables
// module: an inactive customer still pays what it owes.
func (m *Module) CustomerExists(ctx context.Context, tenantID, userID, customerID uuid.UUID) error {
	_, err := m.service.GetCustomer(ctx, app.Actor{TenantID: tenantID, UserID: userID}, customerID)
	return err
}

// CustomerNames returns the names of the customers of tenantID among ids
// (see app.Service.CustomerNames). It is for the receivables module.
func (m *Module) CustomerNames(ctx context.Context, tenantID, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	return m.service.CustomerNames(ctx, app.Actor{TenantID: tenantID, UserID: userID}, ids)
}
