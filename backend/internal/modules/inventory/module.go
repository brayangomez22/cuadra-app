// Package inventory wires the inventory module: locations, stock levels and
// the kardex. internal/server builds it with New and mounts its routes.
package inventory

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Authorize is the role → permission matrix; it decides who sees costs.
	Authorize auth.Authorizer
	// Variants is the catalog as the inventory needs it; internal/server adapts the
	// catalog module to it.
	Variants       app.ProductVariants
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	// Now is the clock of the use cases; nil is time.Now (the demo
	// generator simulates one).
	Now func() time.Time
}

// Module is the wired inventory module.
type Module struct {
	service   *app.Service
	stock     *postgres.StockRepository
	authorize auth.Authorizer
	log       *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Authorize == nil || cfg.Variants == nil || cfg.Logger == nil {
		return nil, errors.New("inventory: missing module dependency")
	}
	stock := postgres.NewStockRepository(cfg.DB)
	svc, err := app.NewService(app.Deps{
		Locations:      postgres.NewLocationRepository(cfg.DB),
		Stock:          stock,
		Policies:       postgres.NewPolicyRepository(cfg.DB),
		Reorder:        postgres.NewReorderRepository(cfg.DB),
		Counts:         postgres.NewStockCountRepository(cfg.DB),
		CountEntries:   postgres.NewCountEntryRepository(cfg.DB),
		Variants:       cfg.Variants,
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Now:            cfg.Now,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, stock: stock, authorize: cfg.Authorize, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "inventory") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.authorize, m.log)
}

// IssueSaleStock registers the stock exits of a sale (see
// app.Service.IssueSaleStock). It is for the sales module, through the port
// internal/server adapts; with a ctx from db.InTenantTx it joins the sale's
// transaction. Errors of one variant are *app.VariantStockError.
func (m *Module) IssueSaleStock(ctx context.Context, tenantID, userID uuid.UUID, in app.SaleStockInput) error {
	_, err := m.service.IssueSaleStock(ctx, app.Actor{TenantID: tenantID, UserID: userID}, in)
	return err
}

// ReceivePurchaseStock registers the stock entries of a purchase (see
// app.Service.ReceivePurchaseStock), in line order. It is for the purchases
// module, through the port internal/server adapts; with a ctx from
// db.InTenantTx it joins the purchase's transaction. Errors of one variant
// are *app.VariantStockError.
func (m *Module) ReceivePurchaseStock(ctx context.Context, tenantID, userID uuid.UUID, in app.PurchaseStockInput) ([]*domain.StockMovement, error) {
	return m.service.ReceivePurchaseStock(ctx, app.Actor{TenantID: tenantID, UserID: userID}, in)
}

// ReverseSaleStock gives back the stock of a voided sale (see
// app.Service.ReverseSaleStock), joining the void's transaction like
// IssueSaleStock.
func (m *Module) ReverseSaleStock(ctx context.Context, tenantID, userID, saleID uuid.UUID) error {
	_, err := m.service.ReverseSaleStock(ctx, app.Actor{TenantID: tenantID, UserID: userID}, saleID)
	return err
}

// ReceiveInitialStock registers the initial stock of new variants at a
// location (see app.Service.ReceiveInitialStock). It is for the imports
// module; with a ctx from db.InTenantTx it joins the import's batch.
func (m *Module) ReceiveInitialStock(ctx context.Context, tenantID, userID, locationID uuid.UUID, lines []app.InitialStockLine) error {
	return m.service.ReceiveInitialStock(ctx, app.Actor{TenantID: tenantID, UserID: userID}, locationID, lines)
}

// CheckLocation fails unless id is an active location of tenantID (see
// app.Service.CheckLocation). It is for the imports module.
func (m *Module) CheckLocation(ctx context.Context, tenantID, userID, id uuid.UUID) error {
	return m.service.CheckLocation(ctx, app.Actor{TenantID: tenantID, UserID: userID}, id)
}

// HasMovements reports whether any of variantIDs has a stock movement in
// tenantID. It is for the catalog, which keeps the base unit of a product
// with movements from changing.
func (m *Module) HasMovements(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (bool, error) {
	return m.stock.HasMovements(ctx, tenantID, variantIDs)
}
