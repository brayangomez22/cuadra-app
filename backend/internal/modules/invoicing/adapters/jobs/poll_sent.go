package jobs

import (
	"context"
	"time"

	"github.com/riverqueue/river"
)

// pollInterval is how often the documents waiting for the DIAN are looked
// up. The provider retries them for three hours; ten minutes is soon enough
// for a receipt and light on its rate limit.
const pollInterval = 10 * time.Minute

// PollSentArgs are the arguments of the periodic job that looks up the sent
// documents. It has no tenant: the use case finds them across tenants and
// works each one within its own.
type PollSentArgs struct{}

// Kind names the job in River.
func (PollSentArgs) Kind() string { return "invoicing.poll_sent" }

// Poller is the use case the job runs (*app.Service).
type Poller interface {
	PollSent(ctx context.Context) (int, error)
}

// PollSentWorker works the PollSentArgs jobs.
type PollSentWorker struct {
	river.WorkerDefaults[PollSentArgs]
	Poller Poller
}

// Work looks up the sent documents. An error makes River retry the job; the
// documents already updated are not sent anymore.
func (w *PollSentWorker) Work(ctx context.Context, _ *river.Job[PollSentArgs]) error {
	_, err := w.Poller.PollSent(ctx)
	return err
}

// PollSentSchedule is the periodic job that enqueues PollSentArgs every
// pollInterval.
func PollSentSchedule() *river.PeriodicJob {
	return river.NewPeriodicJob(river.PeriodicInterval(pollInterval),
		func() (river.JobArgs, *river.InsertOpts) { return PollSentArgs{}, nil },
		&river.PeriodicJobOpts{ID: PollSentArgs{}.Kind()})
}
