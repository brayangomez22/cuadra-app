// Command api runs the Cuadra HTTP API.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
)

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
	log := logger.New(os.Stdout, cfg.LogLevel).With(slog.String("env", cfg.Env))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpx.Healthz)
	mux.HandleFunc("GET /readyz", httpx.Readyz)

	handler := httpx.RequestID(httpx.Logging(log)(httpx.Recover(log)(mux)))
	srv := httpx.NewServer(cfg.HTTPAddr, handler, cfg.ShutdownTimeout)

	log.InfoContext(ctx, "api starting", slog.String("addr", cfg.HTTPAddr))
	if err := srv.Run(ctx); err != nil {
		return err
	}
	log.InfoContext(ctx, "api stopped")
	return nil
}
