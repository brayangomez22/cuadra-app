// Package catalog wires the catalog module: products, categories, units of
// measure, attribute definitions and variant axes. internal/server builds it with
// New and mounts its routes.
package catalog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
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

// Module is the wired catalog module.
type Module struct {
	service   *app.Service
	products  *postgres.ProductRepository
	authorize auth.Authorizer
	log       *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Authorize == nil || cfg.Logger == nil {
		return nil, errors.New("catalog: missing module dependency")
	}
	products := postgres.NewProductRepository(cfg.DB)
	svc, err := app.NewService(app.Deps{
		Products:       products,
		Categories:     postgres.NewCategoryRepository(cfg.DB),
		Units:          postgres.NewUnitRepository(cfg.DB),
		Attributes:     postgres.NewAttributeRepository(cfg.DB),
		VariantAxes:    postgres.NewVariantAxisRepository(cfg.DB),
		Seeds:          postgres.NewSeedRepository(cfg.DB),
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, products: products, authorize: cfg.Authorize, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "catalog") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.authorize, m.log)
}

// SeedTenant gives a new tenant its initial catalog. It is for the sign-up
// (internal/server adapts it to identity's TenantSetup); with a ctx from
// db.InTenantTx it joins the sign-up's transaction.
func (m *Module) SeedTenant(ctx context.Context, tenantID uuid.UUID, seed app.Seed) error {
	return m.service.SeedTenant(ctx, tenantID, seed)
}
