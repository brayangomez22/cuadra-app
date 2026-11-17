// Package jobs runs the feedback module's background job on River: telling
// the team about a comment. It is an adapter like HTTP: it receives the job
// and calls a use case.
package jobs

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// NotifyArgs are the arguments of the job that tells the team about a
// comment: the tenant (set by the API from the token) and the comment's ID,
// never what it says.
type NotifyArgs struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	FeedbackID uuid.UUID `json:"feedback_id"`
}

// Kind names the job in River.
func (NotifyArgs) Kind() string { return "feedback.notify" }

// InsertOpts makes the job unique per comment.
func (NotifyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// Notifier is the use case the job runs (*app.Service).
type Notifier interface {
	NotifyFeedback(ctx context.Context, tenantID, feedbackID uuid.UUID) error
}

// NotifyWorker works the NotifyArgs jobs.
type NotifyWorker struct {
	river.WorkerDefaults[NotifyArgs]
	Notifier Notifier
}

// Work tells the team. An error makes River retry the job later; a comment
// already notified is not notified again.
func (w *NotifyWorker) Work(ctx context.Context, job *river.Job[NotifyArgs]) error {
	return w.Notifier.NotifyFeedback(ctx, job.Args.TenantID, job.Args.FeedbackID)
}

// Enqueuer returns the function the repository enqueues a comment's notice
// with, in the transaction that stores the comment.
func Enqueuer(c *jobs.Client) func(ctx context.Context, tx pgx.Tx, tenantID, feedbackID uuid.UUID) error {
	return func(ctx context.Context, tx pgx.Tx, tenantID, feedbackID uuid.UUID) error {
		_, err := c.InsertTx(ctx, tx, NotifyArgs{TenantID: tenantID, FeedbackID: feedbackID}, nil)
		return err
	}
}
