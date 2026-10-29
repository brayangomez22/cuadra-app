// Command seed generates the demo tenants (T34) in the database of
// DATABASE_URL: for each business template with a demo profile it deletes
// its current demo and writes it again, six months of history ending
// yesterday. The same seed always gives the same data. Its users log in
// with DEMO_PASSWORD.
//
//	go run ./cmd/seed                       # every demo
//	go run ./cmd/seed -template <template> -seed 7
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	template := flag.String("template", "", "business template to generate (default: every demo profile)")
	seed := flag.Uint64("seed", demo.DefaultSeed, "seed of the generated data")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	if len(cfg.DemoPassword) < 12 {
		return errors.New("DEMO_PASSWORD must have at least 12 characters")
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	database, err := db.Open(ctx, cfg.DatabaseURL, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider())
	if err != nil {
		return err
	}
	defer database.Close()
	// The generator enqueues nothing: a client that does not work jobs.
	jobClient, err := jobs.New(jobs.Config{DB: database, Logger: log})
	if err != nil {
		return err
	}

	templates := demo.Templates()
	if *template != "" {
		templates = []string{*template}
	}
	resetter, err := demo.NewResetter(demo.Resetter{
		Generator: demo.Generator{DB: database, Jobs: jobClient, Log: log},
		Templates: templates,
		Password:  cfg.DemoPassword,
		Options:   demo.Options{Seed: *seed},
	})
	if err != nil {
		return err
	}
	for _, t := range templates {
		if err := generate(ctx, resetter, t); err != nil {
			return err
		}
	}
	return nil
}

// generate resets the demo of template and prints how to log in to it.
func generate(ctx context.Context, r *demo.Resetter, template string) error {
	start := time.Now()
	fmt.Printf("%s: generating...\n", template)
	tenant, err := r.Reset(ctx, template)
	if err != nil {
		return fmt.Errorf("%s: %w", template, err)
	}
	p, err := demo.LoadProfile(template)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %s (tenant %s) in %s\n", template, p.Tenant.Name, tenant, time.Since(start).Round(time.Second))
	fmt.Printf("  users: %s", p.OwnerEmail())
	for _, u := range p.Staff() {
		fmt.Printf(", %s", u.Email)
	}
	fmt.Println(" (password: DEMO_PASSWORD)")
	return nil
}
