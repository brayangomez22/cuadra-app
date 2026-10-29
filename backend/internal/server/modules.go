package server

import (
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	modulesadapter "github.com/brayangomez22/cuadra-app/backend/internal/platform/modules/adapters/http"
	modulespg "github.com/brayangomez22/cuadra-app/backend/internal/platform/modules/adapters/postgres"
)

// newModuleService builds the service that turns registry's modules on and
// off per tenant, on PostgreSQL.
func newModuleService(registry *modules.Registry, cfg Config) (*modules.Service, error) {
	return modules.NewService(modules.Deps{
		Registry:       registry,
		Store:          modulespg.NewStore(cfg.DB),
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Log,
		Now:            cfg.Now,
	})
}

// moduleRoutes mounts the operations of the "modules" tag.
func moduleRoutes(svc *modules.Service, log *slog.Logger) func(*http.ServeMux) {
	return func(mux *http.ServeMux) { modulesadapter.Register(mux, svc, log) }
}
