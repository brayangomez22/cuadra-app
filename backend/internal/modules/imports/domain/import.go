// Package domain holds the import of a tenant's initial catalog from a
// spreadsheet: the file's layout, its rows and how they group into
// products, and the import's lifecycle. Every rule here is the same for
// every business; the columns come from the tenant's configuration.
package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Limits of an import file.
const (
	MaxFileBytes = 10 << 20
	MaxRows      = 20_000
	// BatchRows is about how many rows a batch takes (a product is never
	// split, so a batch may take a few more).
	BatchRows = 500
)

// Status is where an import is in its lifecycle.
type Status string

// The statuses: rejected and validated after the upload; queued after the
// confirmation; running while the job works it; completed or failed.
const (
	StatusRejected  Status = "rejected"
	StatusValidated Status = "validated"
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Format is the kind of file.
type Format string

// The formats an import reads.
const (
	FormatXLSX Format = "xlsx"
	FormatCSV  Format = "csv"
)

// ExistingPolicy is what an import does with a product whose SKU the
// catalog already has: update it or leave it as it is.
type ExistingPolicy string

// The policies.
const (
	UpdateExisting ExistingPolicy = "update"
	SkipExisting   ExistingPolicy = "skip"
)

// Counts are an import's totals. Rows and Products are the file's; Created,
// Updated and Skipped are products, counted as the batches are written.
type Counts struct {
	Rows     int
	Products int
	Created  int
	Updated  int
	Skipped  int
	Errors   int
}

// BatchResult is what one batch did.
type BatchResult struct {
	Created int
	Updated int
	Skipped int
}

// Import is the load of a file into the catalog and the stock of a
// location. It is validated on upload and written later, by batches, by a
// background job.
type Import struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	createdBy   uuid.UUID
	locationID  uuid.UUID
	fileName    string
	format      Format
	policy      ExistingPolicy
	status      Status
	counts      Counts
	batchesDone int
	createdAt   time.Time
	startedAt   *time.Time
	finishedAt  *time.Time
}

// NewImportParams are the data of an uploaded file and its validation.
type NewImportParams struct {
	TenantID   uuid.UUID
	UserID     uuid.UUID
	LocationID uuid.UUID
	FileName   string
	Format     Format
	Policy     ExistingPolicy
	Rows       int
	Products   int
	Errors     int
}

// NewImport creates the import of a validated file, with its own UUID v7:
// validated when the file has rows and no errors, rejected otherwise.
// ErrInvalidImport if a datum is missing or out of range.
func NewImport(p NewImportParams, now time.Time) (*Import, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	status := StatusValidated
	if p.Rows == 0 || p.Errors > 0 {
		status = StatusRejected
	}
	return newImport(ImportSnapshot{
		ID: id, TenantID: p.TenantID, CreatedBy: p.UserID, LocationID: p.LocationID,
		FileName: p.FileName, Format: p.Format, Policy: p.Policy, Status: status,
		Counts:    Counts{Rows: p.Rows, Products: p.Products, Errors: p.Errors},
		CreatedAt: now.UTC(),
	})
}

// maxFileName is the longest file name an import keeps.
const maxFileName = 255

// newImport builds an import from its fields, checking its invariants.
func newImport(s ImportSnapshot) (*Import, error) {
	s.FileName = strings.TrimSpace(s.FileName)
	switch {
	case slices.Contains([]uuid.UUID{s.ID, s.TenantID, s.CreatedBy, s.LocationID}, uuid.Nil),
		s.FileName == "", utf8.RuneCountInString(s.FileName) > maxFileName,
		s.Format != FormatXLSX && s.Format != FormatCSV,
		s.Policy != UpdateExisting && s.Policy != SkipExisting,
		!validStatus(s.Status), !validCounts(s.Counts), s.BatchesDone < 0:
		return nil, ErrInvalidImport
	}
	return &Import{
		id: s.ID, tenantID: s.TenantID, createdBy: s.CreatedBy, locationID: s.LocationID,
		fileName: s.FileName, format: s.Format, policy: s.Policy, status: s.Status,
		counts: s.Counts, batchesDone: s.BatchesDone, createdAt: s.CreatedAt,
		startedAt: s.StartedAt, finishedAt: s.FinishedAt,
	}, nil
}

func validStatus(s Status) bool {
	return slices.Contains([]Status{
		StatusRejected, StatusValidated, StatusQueued, StatusRunning, StatusCompleted, StatusFailed,
	}, s)
}

func validCounts(c Counts) bool {
	values := []int{c.Rows, c.Products, c.Created, c.Updated, c.Skipped, c.Errors}
	return c.Rows <= MaxRows && !slices.ContainsFunc(values, func(n int) bool { return n < 0 })
}

// Confirm queues a validated import; ErrImportNotValidated otherwise.
func (i *Import) Confirm() error {
	if i.status != StatusValidated {
		return ErrImportNotValidated
	}
	i.status = StatusQueued
	return nil
}

// Start marks a queued import running. A running one stays running, so a
// retried job goes on from its checkpoint; ErrImportFinished if it already
// ended, ErrImportNotQueued if it was never confirmed.
func (i *Import) Start(now time.Time) error {
	switch i.status {
	case StatusRunning:
		return nil
	case StatusQueued:
		started := now.UTC()
		i.status, i.startedAt = StatusRunning, &started
		return nil
	case StatusCompleted, StatusFailed:
		return ErrImportFinished
	default:
		return ErrImportNotQueued
	}
}

// RecordBatch adds a written batch to the counts and moves the checkpoint;
// ErrImportNotRunning unless the import is running.
func (i *Import) RecordBatch(r BatchResult) error {
	if i.status != StatusRunning {
		return ErrImportNotRunning
	}
	i.counts.Created += r.Created
	i.counts.Updated += r.Updated
	i.counts.Skipped += r.Skipped
	i.batchesDone++
	return nil
}

// Complete ends a running import; ErrImportNotRunning otherwise.
func (i *Import) Complete(now time.Time) error { return i.finish(StatusCompleted, now) }

// Fail ends a running import with the errors of the batch that failed;
// ErrImportNotRunning otherwise.
func (i *Import) Fail(now time.Time, errors int) error {
	if err := i.finish(StatusFailed, now); err != nil {
		return err
	}
	i.counts.Errors += max(errors, 0)
	return nil
}

func (i *Import) finish(status Status, now time.Time) error {
	if i.status != StatusRunning {
		return ErrImportNotRunning
	}
	finished := now.UTC()
	i.status, i.finishedAt = status, &finished
	return nil
}

// ID returns the import's id (UUID v7).
func (i *Import) ID() uuid.UUID { return i.id }

// TenantID returns the tenant the import belongs to.
func (i *Import) TenantID() uuid.UUID { return i.tenantID }

// CreatedBy returns the user who uploaded the file; the job writes as them.
func (i *Import) CreatedBy() uuid.UUID { return i.createdBy }

// LocationID returns the location that receives the initial stock.
func (i *Import) LocationID() uuid.UUID { return i.locationID }

// FileName returns the uploaded file's name.
func (i *Import) FileName() string { return i.fileName }

// Format returns the kind of file.
func (i *Import) Format() Format { return i.format }

// Policy returns what the import does with existing SKUs.
func (i *Import) Policy() ExistingPolicy { return i.policy }

// Status returns where the import is in its lifecycle.
func (i *Import) Status() Status { return i.status }

// Counts returns the import's totals.
func (i *Import) Counts() Counts { return i.counts }

// BatchesDone returns how many batches are written: the checkpoint a
// retried job goes on from.
func (i *Import) BatchesDone() int { return i.batchesDone }

// CreatedAt returns when the file was uploaded, in UTC.
func (i *Import) CreatedAt() time.Time { return i.createdAt }

// StartedAt returns when the job started it, or nil.
func (i *Import) StartedAt() *time.Time { return i.startedAt }

// FinishedAt returns when it completed or failed, or nil.
func (i *Import) FinishedAt() *time.Time { return i.finishedAt }
