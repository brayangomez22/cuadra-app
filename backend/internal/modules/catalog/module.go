// Package catalog wires the catalog module: products and categories. cmd/api
// builds it with New and mounts its routes.
package catalog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/google/uuid"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
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

// MaxProductsByID is the most products ProductsByID returns in one call.
const MaxProductsByID = 100

// ProductsByID returns the products of tenantID among ids (at most
// MaxProductsByID, distinct), active or not; the ones that do not exist in
// the tenant are left out. It is for other modules (sales reads names,
// prices and IVA through it); cmd/api adapts it to their ports. The caller
// already checked the user's permissions for its own operation.
func (m *Module) ProductsByID(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]*domain.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > MaxProductsByID {
		return nil, errors.New("catalog: too many products requested")
	}
	page, err := m.service.SearchProducts(ctx, app.Actor{TenantID: tenantID}, domain.ProductFilter{IDs: ids, Limit: len(ids)})
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// ProductsOfVariants returns the products that the variants among
// variantIDs (at most MaxProductsByID, distinct) belong to in tenantID, with
// all their variants; the variants that do not exist in the tenant are left
// out. It is for other modules that sell variants (sales reads their names,
// labels, prices and IVA through it).
func (m *Module) ProductsOfVariants(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) ([]*domain.Product, error) {
	if len(variantIDs) > MaxProductsByID {
		return nil, errors.New("catalog: too many variants requested")
	}
	productOf, err := m.VariantProducts(ctx, tenantID, variantIDs)
	if err != nil {
		return nil, err
	}
	productIDs := make([]uuid.UUID, 0, len(productOf))
	for _, id := range productOf {
		if !slices.Contains(productIDs, id) {
			productIDs = append(productIDs, id)
		}
	}
	return m.ProductsByID(ctx, tenantID, productIDs)
}

// VariantProducts returns the product of each of variantIDs that exists in
// tenantID; the others are left out. It is for other modules that only know
// the variant (the inventory fills its deprecated product_id with it).
func (m *Module) VariantProducts(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	return m.products.VariantProducts(ctx, tenantID, variantIDs)
}
