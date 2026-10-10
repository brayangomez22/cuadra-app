// Command api runs the Cuadra HTTP API.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/telemetry"
)

const serviceName = "cuadra-api"

// telemetryShutdownTimeout bounds the final flush of pending telemetry.
const telemetryShutdownTimeout = 5 * time.Second

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

	log := logger.New(os.Stdout, cfg.LogLevel, serviceName, httpx.RequestIDAttrs).With(slog.String("env", cfg.Env))

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

	// The API docs are for developers only; other environments do not serve them.
	handler, err := newHandler(log, otel.GetTracerProvider(), database, cfg.Env == "development")
	if err != nil {
		return err
	}
	srv := httpx.NewServer(cfg.HTTPAddr, handler, cfg.ShutdownTimeout)

	log.InfoContext(ctx, "api starting", slog.String("addr", cfg.HTTPAddr))
	if err := srv.Run(ctx); err != nil {
		return err
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
