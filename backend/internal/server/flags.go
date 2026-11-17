package server

import (
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
	flagsadapter "github.com/brayangomez22/cuadra-app/backend/internal/platform/flags/adapters/http"
	flagspg "github.com/brayangomez22/cuadra-app/backend/internal/platform/flags/adapters/postgres"
)

// newFlagService builds the feature flags on PostgreSQL: the evaluator,
// with its cache, and the operators' service.
func newFlagService(cfg Config) (*flags.Service, error) {
	registry, err := flags.NewRegistry(flags.Builtin()...)
	if err != nil {
		return nil, err
	}
	eval, err := flags.NewEvaluator(flags.EvaluatorDeps{
		Registry: registry,
		Store:    flagspg.NewStore(cfg.DB),
		TTL:      cfg.FlagsCacheTTL,
		Now:      cfg.Now,
		Logger:   cfg.Log,
	})
	if err != nil {
		return nil, err
	}
	return flags.NewService(flags.ServiceDeps{
		Evaluator:      eval,
		Operators:      cfg.PlatformOperators,
		Now:            cfg.Now,
		TracerProvider: cfg.TracerProvider,
		MeterProvider:  cfg.MeterProvider,
		Logger:         cfg.Log,
	})
}

// flagRoutes mounts the operations of the "flags" tag.
func flagRoutes(svc *flags.Service, log *slog.Logger) func(*http.ServeMux) {
	return func(mux *http.ServeMux) { flagsadapter.Register(mux, svc, log) }
}
