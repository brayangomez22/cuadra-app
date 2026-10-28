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

// StockCountRepository implements domain.StockCountRepository on
// stock_counts and stock_count_lines.
type StockCountRepository struct {
	db *db.DB
}

// NewStockCountRepository returns a StockCountRepository on d.
func NewStockCountRepository(d *db.DB) *StockCountRepository {
	return &StockCountRepository{db: d}
}

// Start stores the count and takes its snapshot in one transaction. The
// partial unique index on open counts reports another open count at the
// location.
func (r *StockCountRepository) Start(ctx context.Context, c *domain.StockCount, variantIDs []uuid.UUID) (int, error) {
	var lines int64
	err := r.db.WithTenantTx(ctx, c.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.InsertStockCount(ctx, sqlcgen.InsertStockCountParams{
			ID: c.ID(), TenantID: c.TenantID(), LocationID: c.LocationID(), CategoryIds: c.CategoryIDs(),
			Mode: string(c.Mode()), StartedBy: c.StartedBy(), StartedAt: c.StartedAt(),
		}); err != nil {
			return err
		}
		var err error
		if c.IsPartial() {
			lines, err = q.SnapshotVariants(ctx, sqlcgen.SnapshotVariantsParams{
				TenantID: c.TenantID(), CountID: c.ID(), LocationID: c.LocationID(),
				StartedAt: c.StartedAt(), VariantIds: variantIDs,
			})
			return err
		}
		lines, err = q.SnapshotLevels(ctx, sqlcgen.SnapshotLevelsParams{
			CountID: c.ID(), LocationID: c.LocationID(), StartedAt: c.StartedAt(),
		})
		return err
	})
	switch {
	case db.Violates(err, db.UniqueViolation, stockCountsOneOpen):
		return 0, domain.ErrCountInProgress
	case db.Violates(err, db.ForeignKeyViolation, stockCountsLocationFKey):
		return 0, domain.ErrLocationNotFound
	case err != nil:
		return 0, fmt.Errorf("inventory: start stock count: %w", err)
	}
	return int(lines), nil
}

// Get returns the count with id.
func (r *StockCountRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockCount, error) {
	var c *domain.StockCount
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		c, err = readCount(ctx, sqlcgen.New(tx), id)
		return err
	})
	if err != nil && !errors.Is(err, domain.ErrCountNotFound) {
		return nil, fmt.Errorf("inventory: get stock count: %w", err)
	}
	return c, err
}

// readCount reads and rebuilds the count with id; ErrCountNotFound if
// there is none.
func readCount(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (*domain.StockCount, error) {
	row, err := q.GetStockCount(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCountNotFound
	}
	if err != nil {
		return nil, err
	}
	return rehydrateCount(row)
}

// List returns one page of counts and how many match, read in the same
// transaction.
func (r *StockCountRepository) List(ctx context.Context, tenantID uuid.UUID, f domain.CountFilter) (domain.StockCountPage, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.StockCountPage{}, err
	}
	var status *string
	if f.Status != nil {
		s := string(*f.Status)
		status = &s
	}
	var rows []sqlcgen.ListStockCountsRow
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListStockCounts(ctx, sqlcgen.ListStockCountsParams{
			LocationID: f.LocationID, Status: status, RowLimit: limit, RowOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountStockCounts(ctx, sqlcgen.CountStockCountsParams{LocationID: f.LocationID, Status: status})
		return err
	})
	if err != nil {
		return domain.StockCountPage{}, fmt.Errorf("inventory: list stock counts: %w", err)
	}
	page := domain.StockCountPage{Items: make([]*domain.StockCount, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		// Both queries select the same columns.
		c, err := rehydrateCount(sqlcgen.GetStockCountRow(row))
		if err != nil {
			return domain.StockCountPage{}, err
		}
		page.Items = append(page.Items, c)
	}
	return page, nil
}

// Progress counts the lines of the count and those counted in their round.
func (r *StockCountRepository) Progress(ctx context.Context, tenantID, id uuid.UUID) (domain.CountProgress, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.StockCountProgressRow, error) {
		return sqlcgen.New(tx).StockCountProgress(ctx, id)
	})
	if err != nil {
		return domain.CountProgress{}, fmt.Errorf("inventory: stock count progress: %w", err)
	}
	return domain.CountProgress{Lines: int(row.Lines), Counted: int(row.Counted)}, nil
}

// Recount starts a new round of the lines of variantIDs, all or none.
func (r *StockCountRepository) Recount(ctx context.Context, tenantID, id uuid.UUID, variantIDs []uuid.UUID) error {
	errMissing := errors.New("missing line")
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		n, err := sqlcgen.New(tx).RecountLines(ctx, sqlcgen.RecountLinesParams{CountID: id, VariantIds: variantIDs})
		if err == nil && int(n) != len(distinct(variantIDs)) {
			return errMissing
		}
		return err
	})
	switch {
	case errors.Is(err, errMissing):
		return domain.ErrVariantNotInCount
	case err != nil:
		return fmt.Errorf("inventory: recount stock count lines: %w", err)
	}
	return nil
}

// rehydrateCount rebuilds a stored count; a row the domain rejects is
// corrupt.
func rehydrateCount(row sqlcgen.GetStockCountRow) (*domain.StockCount, error) {
	s := domain.StockCountSnapshot{
		StockCountSpec: domain.StockCountSpec{
			ID: row.ID, TenantID: row.TenantID, LocationID: row.LocationID, CategoryIDs: row.CategoryIds,
			Mode: domain.CountMode(row.Mode), StartedBy: row.StartedBy, StartedAt: row.StartedAt,
		},
		Status: domain.CountStatus(row.Status),
	}
	if row.ClosedBy != nil && row.ClosedAt != nil {
		s.ClosedBy, s.ClosedAt = *row.ClosedBy, *row.ClosedAt
	}
	if s.Status == domain.CountApproved {
		s.Summary = &domain.CountSummary{
			Lines: int(row.LinesTotal), Counted: int(row.LinesCounted), Exact: int(row.LinesExact),
			NetValue: row.NetDifferenceValue, AbsoluteValue: row.AbsoluteDifferenceValue,
		}
	}
	c, err := domain.RehydrateStockCount(s)
	if err != nil {
		return nil, fmt.Errorf("inventory: stored stock count is invalid: %w", err)
	}
	return c, nil
}

// distinct returns ids without repetitions.
func distinct(ids []uuid.UUID) map[uuid.UUID]struct{} {
	set := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}
