package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// CountEntryRepository implements domain.CountEntryRepository on
// stock_count_entries.
type CountEntryRepository struct {
	db *db.DB
}

// NewCountEntryRepository returns a CountEntryRepository on d.
func NewCountEntryRepository(d *db.DB) *CountEntryRepository {
	return &CountEntryRepository{db: d}
}

// Add stores the entry in one transaction: it holds the count shared (an
// approval waits for it, and it waits for an approval) and checks it is still
// open, adds the line if asked, and inserts the entry with the line's round.
func (r *CountEntryRepository) Add(ctx context.Context, e *domain.CountEntry, addLine bool) error {
	err := r.db.WithTenantTx(ctx, e.TenantID(), func(tx pgx.Tx) error {
		return addEntry(ctx, sqlcgen.New(tx), e, addLine)
	})
	switch {
	case errors.Is(err, domain.ErrCountNotOpen), errors.Is(err, domain.ErrVariantNotInCount):
		return err
	case errors.Is(err, pgx.ErrNoRows):
		return domain.ErrCountNotFound
	case db.Violates(err, db.ForeignKeyViolation, countLinesVariantFKey):
		return domain.ErrVariantNotFound
	case err != nil:
		return fmt.Errorf("inventory: add stock count entry: %w", err)
	}
	return nil
}

// addEntry runs the steps of Add in its transaction.
func addEntry(ctx context.Context, q *sqlcgen.Queries, e *domain.CountEntry, addLine bool) error {
	status, err := q.ShareStockCount(ctx, e.CountID())
	if err != nil {
		return err
	}
	if domain.CountStatus(status) != domain.CountOpen {
		return domain.ErrCountNotOpen
	}
	if addLine {
		if err := q.AddCountLine(ctx, sqlcgen.AddCountLineParams{
			TenantID: e.TenantID(), CountID: e.CountID(), VariantID: e.VariantID(),
		}); err != nil {
			return err
		}
	}
	n, err := q.InsertCountEntry(ctx, sqlcgen.InsertCountEntryParams{
		ID: e.ID(), CountID: e.CountID(), VariantID: e.VariantID(),
		Quantity: e.Quantity(), CountedBy: e.CountedBy(), CountedAt: e.CountedAt(),
	})
	if err == nil && n == 0 {
		return domain.ErrVariantNotInCount
	}
	return err
}

// Get returns the entry with id in the count.
func (r *CountEntryRepository) Get(ctx context.Context, tenantID, countID, id uuid.UUID) (*domain.CountEntry, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.GetCountEntryRow, error) {
		return sqlcgen.New(tx).GetCountEntry(ctx, sqlcgen.GetCountEntryParams{ID: id, CountID: countID})
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrCountEntryNotFound
	case err != nil:
		return nil, fmt.Errorf("inventory: get stock count entry: %w", err)
	}
	return rehydrateEntry(domain.CountEntrySnapshot(row))
}

// Delete removes the entry.
func (r *CountEntryRepository) Delete(ctx context.Context, e *domain.CountEntry) error {
	n, err := db.Affected(ctx, r.db, e.TenantID(), func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).DeleteCountEntry(ctx, e.ID())
	})
	switch {
	case err != nil:
		return fmt.Errorf("inventory: delete stock count entry: %w", err)
	case n == 0:
		return domain.ErrCountEntryNotFound
	}
	return nil
}

// List returns one page of the entries of the current rounds and how many
// match, read in the same transaction.
func (r *CountEntryRepository) List(ctx context.Context, tenantID uuid.UUID, f domain.EntryFilter) (domain.CountEntryPage, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.CountEntryPage{}, err
	}
	var rows []sqlcgen.ListCountEntriesRow
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListCountEntries(ctx, sqlcgen.ListCountEntriesParams{
			CountID: f.CountID, VariantID: f.VariantID, CountedBy: f.CountedBy, RowLimit: limit, RowOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountCountEntries(ctx, sqlcgen.CountCountEntriesParams{
			CountID: f.CountID, VariantID: f.VariantID, CountedBy: f.CountedBy,
		})
		return err
	})
	if err != nil {
		return domain.CountEntryPage{}, fmt.Errorf("inventory: list stock count entries: %w", err)
	}
	page := domain.CountEntryPage{Items: make([]*domain.CountEntry, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		e, err := rehydrateEntry(domain.CountEntrySnapshot(row))
		if err != nil {
			return domain.CountEntryPage{}, err
		}
		page.Items = append(page.Items, e)
	}
	return page, nil
}

// rehydrateEntry rebuilds a stored entry; a row the domain rejects is
// corrupt.
func rehydrateEntry(s domain.CountEntrySnapshot) (*domain.CountEntry, error) {
	e, err := domain.RehydrateCountEntry(s)
	if err != nil {
		return nil, fmt.Errorf("inventory: stored stock count entry is invalid: %w", err)
	}
	return e, nil
}
