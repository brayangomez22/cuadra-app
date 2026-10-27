// Package postgres implements the imports repository on PostgreSQL. Every
// operation runs in db.WithTenantTx, so Row-Level Security limits it to one
// tenant even if a query forgets its filter.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Enqueue inserts, in tx, the job that writes the import importID of
// tenantID. The jobs adapter provides it, so this package knows no queue.
type Enqueue func(ctx context.Context, tx pgx.Tx, tenantID, importID uuid.UUID) error

// ImportRepository implements domain.ImportRepository.
type ImportRepository struct {
	db      *db.DB
	enqueue Enqueue
}

// NewImportRepository returns an ImportRepository on d that enqueues the
// confirmed imports with enqueue.
func NewImportRepository(d *db.DB, enqueue Enqueue) *ImportRepository {
	return &ImportRepository{db: d, enqueue: enqueue}
}

// Create stores a new import with its file and its errors.
func (r *ImportRepository) Create(ctx context.Context, imp *domain.Import, file []byte, errs []domain.RowError) error {
	c := imp.Counts()
	counts, err := int32s(c.Rows, c.Products, c.Errors)
	if err != nil {
		return err
	}
	err = r.db.WithTenantTx(ctx, imp.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.CreateImport(ctx, sqlcgen.CreateImportParams{
			ID: imp.ID(), TenantID: imp.TenantID(), CreatedBy: imp.CreatedBy(), LocationID: imp.LocationID(),
			FileName: imp.FileName(), Format: string(imp.Format()), Policy: string(imp.Policy()),
			Status: string(imp.Status()), TotalRows: counts[0], TotalProducts: counts[1], Errors: counts[2],
			File: file, CreatedAt: imp.CreatedAt(),
		}); err != nil {
			return err
		}
		return addErrors(ctx, q, imp, errs)
	})
	if err != nil {
		return fmt.Errorf("imports: create import: %w", err)
	}
	return nil
}

// Get returns the import without its file.
func (r *ImportRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Import, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.GetImportRow, error) {
		return sqlcgen.New(tx).GetImport(ctx, sqlcgen.GetImportParams{TenantID: tenantID, ID: id})
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrImportNotFound
	case err != nil:
		return nil, fmt.Errorf("imports: get import: %w", err)
	}
	return importFromRow(row)
}

// File returns the import's file.
func (r *ImportRepository) File(ctx context.Context, tenantID, id uuid.UUID) ([]byte, error) {
	file, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]byte, error) {
		return sqlcgen.New(tx).GetImportFile(ctx, sqlcgen.GetImportFileParams{TenantID: tenantID, ID: id})
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrImportNotFound
	case err != nil:
		return nil, fmt.Errorf("imports: get import file: %w", err)
	}
	return file, nil
}

// Errors returns one page of the import's row errors, by row, and how many
// it has; domain.ErrImportNotFound if the import does not exist.
func (r *ImportRepository) Errors(ctx context.Context, tenantID, id uuid.UUID, limit, offset int) ([]domain.RowError, int, error) {
	if _, err := r.Get(ctx, tenantID, id); err != nil {
		return nil, 0, err
	}
	limit32, offset32, err := db.PageBounds(limit, offset)
	if err != nil {
		return nil, 0, err
	}
	var rows []sqlcgen.ListImportErrorsRow
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListImportErrors(ctx, sqlcgen.ListImportErrorsParams{
			TenantID: tenantID, ImportID: id, Limit: limit32, Offset: offset32,
		}); err != nil {
			return err
		}
		total, err = q.CountImportErrors(ctx, sqlcgen.CountImportErrorsParams{TenantID: tenantID, ImportID: id})
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("imports: list import errors: %w", err)
	}
	out := make([]domain.RowError, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RowError{
			Row: int(row.RowNumber), Column: row.ColumnName, Code: row.Code, Message: row.Message,
		})
	}
	return out, int(total), nil
}

// Queue stores the confirmed import and enqueues its job in the same
// transaction (transactional outbox): if either fails, neither happens.
func (r *ImportRepository) Queue(ctx context.Context, imp *domain.Import) error {
	err := r.db.WithTenantTx(ctx, imp.TenantID(), func(tx pgx.Tx) error {
		if err := save(ctx, sqlcgen.New(tx), imp); err != nil {
			return err
		}
		return r.enqueue(ctx, tx, imp.TenantID(), imp.ID())
	})
	if err != nil {
		return fmt.Errorf("imports: queue import: %w", err)
	}
	return nil
}

// Save stores the import's status, counts and checkpoint, and adds errs.
// With a ctx from db.InTenantTx it joins that transaction.
func (r *ImportRepository) Save(ctx context.Context, imp *domain.Import, errs []domain.RowError) error {
	err := r.db.WithTenantTx(ctx, imp.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := save(ctx, q, imp); err != nil {
			return err
		}
		return addErrors(ctx, q, imp, errs)
	})
	if err != nil {
		return fmt.Errorf("imports: save import: %w", err)
	}
	return nil
}
