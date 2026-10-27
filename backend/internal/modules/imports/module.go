// Package imports wires the imports module: the initial load of a tenant's
// catalog and stock from a spreadsheet. internal/server builds it with New,
// mounts its routes and gives it the catalog, the inventory and the
// activable modules through ports.
package imports

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/http"
	jobsadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/sheet"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Jobs enqueues the confirmed imports; Workers is the bundle of the
	// process's job workers, where the module adds its own.
	Jobs    *jobs.Client
	Workers *river.Workers
	// Catalog, Stock and Modules are the other modules as the imports
	// need them; internal/server adapts them.
	Catalog        app.Catalog
	Stock          app.Stock
	Modules        app.Modules
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Module is the wired imports module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL and adds its worker to cfg.Workers.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Jobs == nil || cfg.Workers == nil || cfg.Logger == nil {
		return nil, errors.New("imports: missing module dependency")
	}
	svc, err := app.NewService(app.Deps{
		Imports:        postgres.NewImportRepository(cfg.DB, jobsadapter.Enqueuer(cfg.Jobs)),
		Tx:             postgres.NewTransactor(cfg.DB),
		Catalog:        cfg.Catalog,
		Stock:          cfg.Stock,
		Modules:        cfg.Modules,
		Sheets:         sheet.Sheets{},
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	if err := river.AddWorkerSafely(cfg.Workers, &jobsadapter.RunImportWorker{Runner: svc}); err != nil {
		return nil, err
	}
	return &Module{service: svc, log: cfg.Logger}, nil
}

// RegisterRoutes mounts the module's operations (tag "imports") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}
