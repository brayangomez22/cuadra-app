// Package jobs runs the invoicing module's background jobs on River: the
// issue of each electronic document and the periodic look up of those the
// DIAN has not answered. It is an adapter like HTTP: it receives the job and
// calls a use case.
package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// Retries of an issue while the provider is down: from 10 seconds, doubling
// up to an hour, for about a day. After the last one the document stays
// pending and a person sends it again from the documents' panel.
const (
	issueMaxAttempts = 30
	firstRetry       = 10 * time.Second
	maxRetry         = time.Hour
	issueTimeout     = time.Minute
)

// IssueDocumentArgs are the arguments of the job that issues a document.
// The tenant travels here because the job has no request; it was set by the
// sale's confirmation, from the token.
type IssueDocumentArgs struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	DocumentID uuid.UUID `json:"document_id"`
}

// Kind names the job in River.
func (IssueDocumentArgs) Kind() string { return "invoicing.issue" }

// InsertOpts makes the job unique per document while it waits or runs, so a
// manual retry of a document still being retried enqueues nothing new.
func (IssueDocumentArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{MaxAttempts: issueMaxAttempts, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// Issuer is the use case the job runs (*app.Service).
type Issuer interface {
	IssueDocument(ctx context.Context, tenantID, id uuid.UUID) error
}

// IssueDocumentWorker works the IssueDocumentArgs jobs.
type IssueDocumentWorker struct {
	river.WorkerDefaults[IssueDocumentArgs]
	Issuer Issuer
}

// Work issues the document. An error (the provider is down) makes River
// retry it later.
func (w *IssueDocumentWorker) Work(ctx context.Context, job *river.Job[IssueDocumentArgs]) error {
	return w.Issuer.IssueDocument(ctx, job.Args.TenantID, job.Args.DocumentID)
}

// Timeout is the most one attempt may take.
func (w *IssueDocumentWorker) Timeout(*river.Job[IssueDocumentArgs]) time.Duration {
	return issueTimeout
}

// NextRetry backs off exponentially from firstRetry up to maxRetry.
func (w *IssueDocumentWorker) NextRetry(job *river.Job[IssueDocumentArgs]) time.Time {
	return time.Now().Add(Backoff(job.Attempt))
}

// Backoff is the wait before retrying after the attempt-th failure.
func Backoff(attempt int) time.Duration {
	wait := firstRetry
	for i := 1; i < attempt && wait < maxRetry; i++ {
		wait *= 2
	}
	return min(wait, maxRetry)
}

// Enqueuer returns the function the repository enqueues a document's issue
// with, in the transaction that stores it.
func Enqueuer(c *jobs.Client) func(ctx context.Context, tx pgx.Tx, tenantID, documentID uuid.UUID) error {
	return func(ctx context.Context, tx pgx.Tx, tenantID, documentID uuid.UUID) error {
		_, err := c.InsertTx(ctx, tx, IssueDocumentArgs{TenantID: tenantID, DocumentID: documentID}, nil)
		return err
	}
}
