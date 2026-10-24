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

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/errreport"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/telemetry"
	"github.com/brayangomez22/cuadra-app/backend/internal/server"
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
	log, flushTelemetry, err := setUpTelemetry(ctx, cfg)
	if err != nil {
		return err
	}
	defer flushTelemetry()

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
	handler, err := newAPIHandler(cfg, database, log)
	if err != nil {
		return err
	}
	return serve(ctx, cfg, log, httpx.NewServer(cfg.HTTPAddr, handler, cfg.ShutdownTimeout), jobClient)
}

// setUpTelemetry starts the traces, metrics and logs, with every Error log
// also sent to Sentry (a no-op without SENTRY_DSN). flush sends what is
// pending: the telemetry first, then the Sentry events, so Sentry gets
// whatever the telemetry's shutdown logs.
func setUpTelemetry(ctx context.Context, cfg config.Config) (log *slog.Logger, flush func(), err error) {
	shutdownTelemetry, err := telemetry.Setup(ctx, telemetry.Resource{
		ServiceName:    serviceName,
		ServiceVersion: version(),
		Environment:    cfg.Env,
	}, os.Getenv)
	if err != nil {
		return nil, nil, err
	}
	reporter, err := errreport.New(errreport.Options{
		DSN:         cfg.SentryDSN,
		Environment: cfg.Env,
		Release:     cmp.Or(cfg.SentryRelease, version()),
		Extractors:  []func(context.Context) []slog.Attr{httpx.RequestIDAttrs, auth.PrincipalAttrs},
	})
	if err != nil {
		return nil, nil, err
	}

	base := logger.New(os.Stdout, cfg.LogLevel, serviceName, httpx.RequestIDAttrs, auth.PrincipalAttrs)
	log = slog.New(reporter.Handler(base.Handler())).With(slog.String("env", cfg.Env))

	// Export failures are degraded telemetry, not app failures. They go only to
	// stdout: sending them through OTLP would loop when the log export fails.
	otelLog := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(slog.String("env", cfg.Env))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		otelLog.WarnContext(context.Background(), "telemetry export failed", slog.Any("error", err))
	}))
	flush = func() {
		// ctx is canceled by now; the flush gets its own deadline.
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), telemetryShutdownTimeout)
		defer cancel()
		if err := shutdownTelemetry(flushCtx); err != nil {
			otelLog.WarnContext(flushCtx, "telemetry shutdown failed", slog.Any("error", err))
		}
		reporter.Flush(errorFlushTimeout)
	}
	return log, flush, nil
}

// newAPIHandler builds the API (internal/server) with the access tokens of
// cfg.
func newAPIHandler(cfg config.Config, database *db.DB, log *slog.Logger) (http.Handler, error) {
	tokens, err := auth.NewJWT(cfg.JWTSecret, accessTokenTTL)
	if err != nil {
		return nil, err
	}
	return server.New(server.Config{
		DB:     database,
		Tokens: tokens,
		// The API docs are for developers only; other environments do not serve them.
		Docs:           cfg.Env == "development",
		Log:            log,
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
	})
}

// serve runs the API until ctx is canceled, with the job workers.
func serve(ctx context.Context, cfg config.Config, log *slog.Logger, srv *httpx.Server, jobClient *jobs.Client) error {
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
