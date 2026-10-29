// Package purchases wires the purchases module: the tenant's suppliers and
// the purchases registered from their invoices. internal/server builds it
// with New, giving it the catalog's variants and the inventory's entries
// through the ports of app, and mounts its routes.
package purchases

import (
	"errors"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Catalog and Stock are the catalog and inventory modules, adapted by
	// internal/server.
	Catalog        app.VariantCatalog
	Stock          app.PurchaseStock
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Module is the wired purchases module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Logger == nil {
		return nil, errors.New("purchases: missing module dependency")
	}
	svc, err := app.NewService(app.Deps{
		Suppliers:      postgres.NewSupplierRepository(cfg.DB),
		Purchases:      postgres.NewPurchaseRepository(cfg.DB),
		Catalog:        cfg.Catalog,
		Stock:          cfg.Stock,
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "purchases") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}
