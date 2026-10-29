// Package sales wires the sales module: point-of-sale sales, their numbering
// and voiding. internal/server builds it with New, giving it the catalog and the
// stock through the ports of app, and mounts its routes.
package sales

import (
	"errors"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Authorize is the role → permission matrix; it decides who discounts
	// without the tenant's limit.
	Authorize auth.Authorizer
	// Catalog and Stock are the catalog and the inventory, adapted by internal/server.
	Catalog app.VariantCatalog
	Stock   app.StockReserver
	// Customers checks the customer of a sale; internal/server adapts the
	// customers module to it.
	Customers app.CustomerDirectory
	// Credit opens and cancels the receivables of credit sales, and
	// Authorizer checks who lets one exceed the customer's credit;
	// internal/server adapts the receivables and identity modules to them.
	Credit     app.CreditLedger
	Authorizer app.CreditAuthorizer
	// Prices gives the unit price of each line: the chain of the price rules
	// of the active modules. Nil sells each variant at its price.
	Prices         app.PriceResolver
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Module is the wired sales module.
type Module struct {
	service   *app.Service
	authorize auth.Authorizer
	log       *slog.Logger
}

// New builds the module on PostgreSQL.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Authorize == nil || cfg.Logger == nil {
		return nil, errors.New("sales: missing module dependency")
	}
	svc, err := app.NewService(app.Deps{
		Catalog:        cfg.Catalog,
		Prices:         cfg.Prices,
		Stock:          cfg.Stock,
		Customers:      cfg.Customers,
		Credit:         cfg.Credit,
		Authorizer:     cfg.Authorizer,
		Sales:          postgres.NewSaleRepository(cfg.DB),
		Policies:       postgres.NewPolicyRepository(cfg.DB),
		CashSessions:   postgres.NewCashSessionRepository(cfg.DB),
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	return &Module{service: svc, authorize: cfg.Authorize, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "sales") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.authorize, m.log)
}
