// Package invoicing wires the invoicing module: the tenant's emission
// settings and numbering resolutions, the electronic document each
// confirmed sale gets and its issue by the technology provider in the
// background. internal/server builds it with New, giving it the sales and
// the customers through the ports of app, and mounts its routes.
package invoicing

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/http"
	jobsadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/simulated"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// ProviderSimulated names the provider that never reaches the DIAN.
const ProviderSimulated = "simulated"

// Config are the module's dependencies.
type Config struct {
	DB *db.DB
	// Jobs enqueues the documents' issue; Workers is the bundle of the
	// process's job workers, where the module adds its own.
	Jobs    *jobs.Client
	Workers *river.Workers
	// Provider names the technology provider (INVOICING_PROVIDER).
	Provider string
	// Sales and Buyers are the sales and the customers modules, adapted by
	// internal/server.
	Sales          app.SaleSource
	Buyers         app.BuyerDirectory
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	// Now is the clock of the use cases; nil is time.Now.
	Now func() time.Time
	// Schedule adds the module's periodic jobs to Jobs (the API process).
	Schedule bool
}

// Module is the wired invoicing module.
type Module struct {
	service *app.Service
	log     *slog.Logger
}

// New builds the module on PostgreSQL and adds its workers to cfg.Workers.
func New(cfg Config) (*Module, error) {
	if cfg.DB == nil || cfg.Jobs == nil || cfg.Workers == nil || cfg.Logger == nil {
		return nil, errors.New("invoicing: missing module dependency")
	}
	invoicer, err := provider(cfg.Provider)
	if err != nil {
		return nil, err
	}
	svc, err := app.NewService(app.Deps{
		Settings:       postgres.NewSettingsRepository(cfg.DB),
		Resolutions:    postgres.NewResolutionRepository(cfg.DB),
		Documents:      postgres.NewDocumentRepository(cfg.DB, jobsadapter.Enqueuer(cfg.Jobs)),
		Invoicer:       invoicer,
		Sales:          cfg.Sales,
		Buyers:         cfg.Buyers,
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Now:            cfg.Now,
		Logger:         cfg.Logger,
	})
	if err != nil {
		return nil, err
	}
	if err := addJobs(cfg, svc); err != nil {
		return nil, err
	}
	return &Module{service: svc, log: cfg.Logger}, nil
}

// provider returns the adapter of the named technology provider.
func provider(name string) (app.ElectronicInvoicer, error) {
	if name == ProviderSimulated {
		return simulated.Invoicer{}, nil
	}
	return nil, fmt.Errorf("invoicing: unknown provider %q", name)
}

// addJobs adds the issue and poll workers and, in the API process, the poll's
// schedule.
func addJobs(cfg Config, svc *app.Service) error {
	if err := river.AddWorkerSafely(cfg.Workers, &jobsadapter.IssueDocumentWorker{Issuer: svc}); err != nil {
		return err
	}
	if err := river.AddWorkerSafely(cfg.Workers, &jobsadapter.PollSentWorker{Poller: svc}); err != nil {
		return err
	}
	if cfg.Schedule {
		return cfg.Jobs.AddPeriodic(jobsadapter.PollSentSchedule())
	}
	return nil
}

// RegisterRoutes mounts the module's operations (tag "invoicing") on mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	httpadapter.Register(mux, m.service, m.log)
}
