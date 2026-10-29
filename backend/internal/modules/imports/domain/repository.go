package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ImportRepository persists imports with their file and their row errors.
// Every operation is scoped to a tenant: an import of another tenant is
// reported as ErrImportNotFound.
type ImportRepository interface {
	// Create stores a new import with its file and its errors (at most
	// MaxErrors).
	Create(ctx context.Context, imp *Import, file []byte, errs []RowError) error
	// Get returns the import without its file.
	Get(ctx context.Context, tenantID, id uuid.UUID) (*Import, error)
	// File returns the import's file.
	File(ctx context.Context, tenantID, id uuid.UUID) ([]byte, error)
	// Errors returns one page of the import's row errors, by row, and how
	// many it has.
	Errors(ctx context.Context, tenantID, id uuid.UUID, limit, offset int) ([]RowError, int, error)
	// Queue stores the confirmed import and enqueues the job that writes
	// it, in the same transaction: the job exists if and only if the
	// confirmation is stored.
	Queue(ctx context.Context, imp *Import) error
	// Save stores the import's status, counts and checkpoint, and adds errs
	// to its errors. With a ctx from a transaction it joins it, so a batch
	// and its checkpoint are stored together.
	Save(ctx context.Context, imp *Import, errs []RowError) error
	// PurgeableFiles returns, across tenants, at most limit imports that
	// still have their file and ended before before (ids only: the job
	// that purges them runs with no tenant).
	PurgeableFiles(ctx context.Context, before time.Time, limit int) ([]FileRef, error)
	// DropFile deletes the import's file and keeps the import.
	DropFile(ctx context.Context, tenantID, id uuid.UUID) error
}
