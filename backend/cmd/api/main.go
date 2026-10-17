// Command api runs the Cuadra HTTP API.
package main

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/errreport"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/telemetry"
)

const serviceName = "cuadra-api"

// telemetryShutdownTimeout bounds the final flush of pending telemetry.
const telemetryShutdownTimeout = 5 * time.Second

// errorFlushTimeout bounds the final send of pending Sentry events.
const errorFlushTimeout = 2 * time.Second

// accessTokenTTL is the lifetime of a JWT access token. A role change or a
// deactivation reaches tokens already issued only when they expire.
const accessTokenTTL = 15 * time.Minute

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	shutdownTelemetry, err := telemetry.Setup(ctx, telemetry.Resource{
		ServiceName:    serviceName,
		ServiceVersion: version(),
		Environment:    cfg.Env,
	}, os.Getenv)
	if err != nil {
		return err
	}

	// Every Error log also goes to Sentry (a no-op without SENTRY_DSN). Its
	// flush is deferred before the telemetry and database ones, so it runs
	// after them and sends whatever their shutdown logs.
	reporter, err := errreport.New(errreport.Options{
		DSN:         cfg.SentryDSN,
		Environment: cfg.Env,
		Release:     cmp.Or(cfg.SentryRelease, version()),
		Extractors:  []func(context.Context) []slog.Attr{httpx.RequestIDAttrs, auth.PrincipalAttrs},
	})
	if err != nil {
		return err
	}
	defer reporter.Flush(errorFlushTimeout)

	base := logger.New(os.Stdout, cfg.LogLevel, serviceName, httpx.RequestIDAttrs, auth.PrincipalAttrs)
	log := slog.New(reporter.Handler(base.Handler())).With(slog.String("env", cfg.Env))

	// Export failures are degraded telemetry, not app failures. They go only to
	// stdout: sending them through OTLP would loop when the log export fails.
	otelLog := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(slog.String("env", cfg.Env))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		otelLog.Warn("telemetry export failed", slog.Any("error", err))
	}))
	defer func() {
		// ctx is canceled by now; the flush gets its own deadline.
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), telemetryShutdownTimeout)
		defer cancel()
		if err := shutdownTelemetry(flushCtx); err != nil {
			otelLog.WarnContext(flushCtx, "telemetry shutdown failed", slog.Any("error", err))
		}
	}()

	// Fails fast if the database is down or the role could bypass RLS; the
	// orchestrator restarts the process.
	database, err := db.Open(ctx, cfg.DatabaseURL, otel.GetTracerProvider(), otel.GetMeterProvider())
	if err != nil {
		return err
	}
	defer database.Close()

	// Workers run in this process. Modules register theirs here when they add a
	// job kind; until then the client only enqueues.
	jobClient, err := jobs.New(jobs.Config{
		DB:             database,
		Logger:         log,
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
	})
	if err != nil {
		return err
	}

	tokens, err := auth.NewJWT(cfg.JWTSecret, accessTokenTTL)
	if err != nil {
		return err
	}
	// The activable modules and the permission matrix are built before the
	// business modules, which authorize with it.
	registry, err := modules.NewRegistry(modules.Builtin()...)
	if err != nil {
		return err
	}
	moduleService, err := newModuleService(registry, database, log, otel.GetTracerProvider(), otel.GetMeterProvider())
	if err != nil {
		return err
	}
	permissions, err := identity.NewPermissions(identityRegistry{registry})
	if err != nil {
		return err
	}

	catalogModule, err := catalog.New(catalog.Config{
		DB:             database,
		Authorize:      permissions.Authorize,
		Logger:         log,
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
	})
	if err != nil {
		return err
	}
	setup, err := newTenantSetup(registry, catalogModule, moduleService)
	if err != nil {
		return err
	}
	identityModule, err := identity.New(identity.Config{
		DB:             database,
		Issuer:         tokens,
		Setup:          setup,
		Permissions:    permissions,
		Modules:        moduleService,
		Logger:         log,
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
	})
	if err != nil {
		return err
	}

	inventoryModule, err := inventory.New(inventory.Config{
		DB:             database,
		Authorize:      permissions.Authorize,
		Variants:       inventoryCatalog{catalogModule},
		Logger:         log,
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
	})
	if err != nil {
		return err
	}

	salesModule, err := sales.New(sales.Config{
		DB:             database,
		Authorize:      permissions.Authorize,
		Catalog:        salesCatalog{catalogModule},
		Stock:          salesStock{inventoryModule},
		Logger:         log,
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
	})
	if err != nil {
		return err
	}

	handler, err := newHandler(handlerConfig{
		Log:            log,
		TracerProvider: otel.GetTracerProvider(),
		DB:             database,
		// The API docs are for developers only; other environments do not serve them.
		Docs:        cfg.Env == "development",
		Verifier:    tokens,
		Permissions: permissions,
		Modules:     moduleService,
		Routes: []func(*http.ServeMux){
			identityModule.RegisterRoutes, catalogModule.RegisterRoutes, inventoryModule.RegisterRoutes,
			salesModule.RegisterRoutes, moduleRoutes(moduleService, log),
		},
	})
	if err != nil {
		return err
	}
	srv := httpx.NewServer(cfg.HTTPAddr, handler, cfg.ShutdownTimeout)

	if err := jobClient.Start(ctx); err != nil {
		return err
	}
	log.InfoContext(ctx, "api starting", slog.String("addr", cfg.HTTPAddr))
	runErr := srv.Run(ctx)

	// After the HTTP server drains, so no request enqueues while the workers
	// stop. ctx is canceled by now; the stop gets its own deadline.
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := jobClient.Stop(stopCtx); err != nil {
		log.WarnContext(stopCtx, "job workers did not stop cleanly", slog.Any("error", err))
	}
	if runErr != nil {
		return runErr
	}
	log.InfoContext(ctx, "api stopped")
	return nil
}

// version identifies the build: the VCS revision when the binary was built
// from a git checkout, the module version otherwise.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return info.Main.Version
}
