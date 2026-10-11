// Package jobs runs background jobs on River, a queue stored in PostgreSQL.
//
// Jobs are enqueued only inside the transaction of the business change that
// produces them (transactional outbox): if the transaction rolls back the job
// never exists, and a committed change never loses its job.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// maxWorkers is how many jobs the process works at once.
const maxWorkers = 10

// Config configures a Client.
type Config struct {
	DB *db.DB
	// Workers are the job kinds this process works. Without workers the client
	// only enqueues, and Start and Stop do nothing.
	Workers *river.Workers
	// PeriodicJobs are enqueued on a schedule. They need Workers.
	PeriodicJobs []*river.PeriodicJob
	// RetryPolicy decides when a failed job runs again. Nil uses River's
	// exponential backoff with jitter.
	RetryPolicy    river.ClientRetryPolicy
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Client enqueues jobs and, when it has workers, works them.
type Client struct {
	river   *river.Client[pgx.Tx]
	working bool
}

// New builds a Client. It fails if a metric instrument cannot be created or
// River rejects the configuration.
func New(cfg Config) (*Client, error) {
	if cfg.DB == nil {
		return nil, errors.New("jobs: DB is required")
	}
	if cfg.TracerProvider == nil {
		cfg.TracerProvider = tracenoop.NewTracerProvider()
	}
	if cfg.MeterProvider == nil {
		cfg.MeterProvider = metricnoop.NewMeterProvider()
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.DiscardHandler)
	}

	tel, err := newTelemetry(cfg.TracerProvider, cfg.MeterProvider)
	if err != nil {
		return nil, err
	}
	rc := &river.Config{
		Logger:       cfg.Logger,
		Middleware:   []rivertype.Middleware{tel},
		PeriodicJobs: cfg.PeriodicJobs,
		RetryPolicy:  cfg.RetryPolicy,
	}
	working := cfg.Workers != nil
	if working {
		rc.Workers = cfg.Workers
		rc.Queues = map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: maxWorkers}}
	}
	client, err := river.NewClient(riverpgxv5.New(cfg.DB.Pool()), rc)
	if err != nil {
		return nil, fmt.Errorf("jobs: create client: %w", err)
	}
	return &Client{river: client, working: working}, nil
}

// InsertTx enqueues a job in tx, the transaction of the change that produces
// it, and returns the job's id. The job becomes visible to the workers only
// when tx commits. There is deliberately no way to enqueue outside a
// transaction.
func (c *Client) InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (int64, error) {
	res, err := c.river.InsertTx(ctx, tx, args, opts)
	if err != nil {
		return 0, fmt.Errorf("jobs: insert %s: %w", args.Kind(), err)
	}
	return res.Job.ID, nil
}

// Start starts working jobs until Stop is called or ctx is canceled.
func (c *Client) Start(ctx context.Context) error {
	if !c.working {
		return nil
	}
	if err := c.river.Start(ctx); err != nil {
		return fmt.Errorf("jobs: start: %w", err)
	}
	return nil
}

// Stop stops fetching jobs and waits for the running ones to finish, or for
// ctx to end. A job interrupted by shutdown is retried later.
func (c *Client) Stop(ctx context.Context) error {
	if !c.working {
		return nil
	}
	if err := c.river.Stop(ctx); err != nil {
		return fmt.Errorf("jobs: stop: %w", err)
	}
	return nil
}
