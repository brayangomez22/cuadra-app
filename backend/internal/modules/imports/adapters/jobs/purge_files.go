package jobs

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// purgeHour is when the files are purged each night, in Bogotá: after the
// demo reset (3:00) and far from the business hours.
const purgeHour = 4

// PurgeFilesArgs are the arguments of the daily job that deletes the files
// of ended imports. It has no tenant: the use case finds them in every
// tenant and works each one within its own.
type PurgeFilesArgs struct{}

// Kind names the job in River.
func (PurgeFilesArgs) Kind() string { return "imports.purge_files" }

// Purger is the use case the job runs (*app.Service).
type Purger interface {
	PurgeFiles(ctx context.Context) (int, error)
}

// PurgeFilesWorker works the PurgeFilesArgs jobs.
type PurgeFilesWorker struct {
	river.WorkerDefaults[PurgeFilesArgs]
	Purger Purger
}

// Work deletes the expired files. An error makes River retry the job; the
// files already deleted are not found again.
func (w *PurgeFilesWorker) Work(ctx context.Context, _ *river.Job[PurgeFilesArgs]) error {
	_, err := w.Purger.PurgeFiles(ctx)
	return err
}

// PurgeFilesSchedule is the periodic job that enqueues PurgeFilesArgs every
// night at purgeHour in Bogotá.
func PurgeFilesSchedule() *river.PeriodicJob {
	return river.NewPeriodicJob(jobs.Daily(purgeHour, 0, jobs.Bogota),
		func() (river.JobArgs, *river.InsertOpts) { return PurgeFilesArgs{}, nil },
		&river.PeriodicJobOpts{ID: PurgeFilesArgs{}.Kind()})
}
