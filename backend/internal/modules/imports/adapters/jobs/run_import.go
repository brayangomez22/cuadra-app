// Package jobs runs the imports module's background job on River: the
// writing of a confirmed import. It is an adapter like HTTP: it receives
// the job and calls a use case.
package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// runTimeout is how long one attempt may take: an import of 20,000 rows
// takes a few minutes. An attempt cut short is retried from its checkpoint.
const runTimeout = 30 * time.Minute

// RunImportArgs are the arguments of the job that writes an import. The
// tenant travels here because the job has no request: the worker opens the
// tenant's transactions with it. The tenant was set by the API when the
// import was confirmed, never by the client.
type RunImportArgs struct {
	TenantID uuid.UUID `json:"tenant_id"`
	ImportID uuid.UUID `json:"import_id"`
}

// Kind names the job in River.
func (RunImportArgs) Kind() string { return "imports.run" }

// InsertOpts makes the job unique per import: confirming twice never
// enqueues it twice.
func (RunImportArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// Runner is the use case the job runs (*app.Service).
type Runner interface {
	RunImport(ctx context.Context, tenantID, id uuid.UUID) error
}

// RunImportWorker works the RunImportArgs jobs.
type RunImportWorker struct {
	river.WorkerDefaults[RunImportArgs]
	Runner Runner
}

// Work writes the import. An error makes River retry the job later.
func (w *RunImportWorker) Work(ctx context.Context, job *river.Job[RunImportArgs]) error {
	return w.Runner.RunImport(ctx, job.Args.TenantID, job.Args.ImportID)
}

// Timeout is the most one attempt may take.
func (w *RunImportWorker) Timeout(*river.Job[RunImportArgs]) time.Duration { return runTimeout }

// Enqueuer returns the function the repository enqueues a confirmed import
// with, in the transaction that confirms it.
func Enqueuer(c *jobs.Client) func(ctx context.Context, tx pgx.Tx, tenantID, importID uuid.UUID) error {
	return func(ctx context.Context, tx pgx.Tx, tenantID, importID uuid.UUID) error {
		_, err := c.InsertTx(ctx, tx, RunImportArgs{TenantID: tenantID, ImportID: importID}, nil)
		return err
	}
}
