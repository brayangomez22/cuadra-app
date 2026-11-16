// Package jobs holds the identity module's background jobs: the nightly
// purge of expired refresh tokens.
package jobs

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// purgeHour and purgeMinute are when the expired tokens are purged each
// night, in Bogotá: after the demo reset (3:00) and the files' purge (4:00).
const (
	purgeHour   = 4
	purgeMinute = 30
)

// PurgeTokensArgs are the arguments of the daily job that deletes the
// expired refresh tokens. It has no tenant: the use case finds them in
// every tenant and works each one within its own.
type PurgeTokensArgs struct{}

// Kind names the job in River.
func (PurgeTokensArgs) Kind() string { return "identity.purge_refresh_tokens" }

// Purger is the use case the job runs (*app.Service).
type Purger interface {
	PurgeExpiredRefreshTokens(ctx context.Context) (int, error)
}

// PurgeTokensWorker works the PurgeTokensArgs jobs.
type PurgeTokensWorker struct {
	river.WorkerDefaults[PurgeTokensArgs]
	Purger Purger
}

// Work deletes the expired tokens. An error makes River retry the job; the
// tokens already deleted are not found again.
func (w *PurgeTokensWorker) Work(ctx context.Context, _ *river.Job[PurgeTokensArgs]) error {
	_, err := w.Purger.PurgeExpiredRefreshTokens(ctx)
	return err
}

// PurgeTokensSchedule is the periodic job that enqueues PurgeTokensArgs
// every night in Bogotá.
func PurgeTokensSchedule() *river.PeriodicJob {
	return river.NewPeriodicJob(jobs.Daily(purgeHour, purgeMinute, jobs.Bogota),
		func() (river.JobArgs, *river.InsertOpts) { return PurgeTokensArgs{}, nil },
		&river.PeriodicJobOpts{ID: PurgeTokensArgs{}.Kind()})
}
