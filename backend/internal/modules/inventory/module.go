// Package inventory wires the inventory module: locations, stock levels and
// the kardex. cmd/api builds it with New and mounts its routes.
package inventory

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Authorize is the role → permission matrix; it decides who sees costs.
	Authorize      auth.Authorizer
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Module is the wired inventory module.
type Module struct {
	service   *app.Service
	authorize auth.Authorizer
	log       *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Authorize == nil || cfg.Logger == nil {
		return nil, errors.New("inventory: missing module dependency")
	}
	svc, err := app.NewService(app.Deps{
		Locations:      postgres.NewLocationRepository(cfg.DB),
		Stock:          postgres.NewStockRepository(cfg.DB),
		Policies:       postgres.NewPolicyRepository(cfg.DB),
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, authorize: cfg.Authorize, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "inventory") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.authorize, m.log)
}

// IssueSaleStock registers the stock exits of a sale (see
// app.Service.IssueSaleStock). It is for the sales module, through the port
// cmd/api adapts; with a ctx from db.InTenantTx it joins the sale's
// transaction. Errors of one product are *app.ProductStockError.
func (m *Module) IssueSaleStock(ctx context.Context, tenantID, userID uuid.UUID, in app.SaleStockInput) error {
	_, err := m.service.IssueSaleStock(ctx, app.Actor{TenantID: tenantID, UserID: userID}, in)
	return err
}

// ReverseSaleStock gives back the stock of a voided sale (see
// app.Service.ReverseSaleStock), joining the void's transaction like
// IssueSaleStock.
func (m *Module) ReverseSaleStock(ctx context.Context, tenantID, userID, saleID uuid.UUID) error {
	_, err := m.service.ReverseSaleStock(ctx, app.Actor{TenantID: tenantID, UserID: userID}, saleID)
	return err
}
