package main

import (
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	modulesadapter "github.com/brayangomez22/cuadra-app/backend/internal/platform/modules/adapters/http"
	modulespg "github.com/brayangomez22/cuadra-app/backend/internal/platform/modules/adapters/postgres"
)

// newModuleService builds the service that turns registry's modules on and
// off per tenant, on PostgreSQL.
func newModuleService(registry *modules.Registry, database *db.DB, log *slog.Logger, tp trace.TracerProvider, mp metric.MeterProvider) (*modules.Service, error) {
	return modules.NewService(modules.Deps{
		Registry:       registry,
		Store:          modulespg.NewStore(database),
		TracerProvider: tp,
		MeterProvider:  mp,
		Logger:         log,
	})
}

// moduleRoutes mounts the operations of the "modules" tag.
func moduleRoutes(svc *modules.Service, log *slog.Logger) func(*http.ServeMux) {
	return func(mux *http.ServeMux) { modulesadapter.Register(mux, svc, log) }
}
