package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// Sheet reads the lines of c and the totals of their counters in the same
// transaction, so both see the same entries.
func (r *StockCountRepository) Sheet(ctx context.Context, c *domain.StockCount) ([]domain.SheetLine, error) {
	var lines []sqlcgen.StockCountSheetRow
	var totals []sqlcgen.StockCountTotalsRow
	err := r.db.WithTenantTx(ctx, c.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if lines, err = q.StockCountSheet(ctx, sqlcgen.StockCountSheetParams{
			CountID: c.ID(), LocationID: c.LocationID(), StartedAt: c.StartedAt(),
		}); err != nil {
			return err
		}
		totals, err = q.StockCountTotals(ctx, c.ID())
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("inventory: read stock count sheet: %w", err)
	}
	counts := make(map[uuid.UUID][]domain.CounterTotal, len(totals))
	for _, t := range totals {
		counts[t.VariantID] = append(counts[t.VariantID], domain.CounterTotal{UserID: t.CountedBy, Quantity: t.Quantity})
	}
	sheet := make([]domain.SheetLine, 0, len(lines))
	for _, l := range lines {
		sheet = append(sheet, domain.SheetLine{
			CountLine: domain.CountLine{
				VariantID: l.VariantID, StartQuantity: l.StartQuantity, StartCost: l.StartCost, Round: int(l.Round),
			},
			Counts:    counts[l.VariantID],
			Movements: l.Movements,
		})
	}
	return sheet, nil
}

// Close runs fn and stores the count's new state in one transaction:
//
//  1. locks the count (SELECT ... FOR UPDATE): an entry being added holds
//     it shared, so fn reads every entry committed before and none arrives
//     until the commit, when Add finds the count closed;
//  2. reads it and calls fn with a ctx that carries the transaction, so the
//     sheet fn reads and the stock it adjusts join it;
//  3. stores the status fn left (approved with its summary, or canceled).
func (r *StockCountRepository) Close(ctx context.Context, tenantID, id uuid.UUID, fn func(context.Context, *domain.StockCount) error) error {
	var fnErr error
	err := r.db.InTenantTx(ctx, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if _, err := q.LockStockCount(ctx, id); err != nil {
			return err
		}
		c, err := readCount(ctx, q, id)
		if err != nil {
			return err
		}
		if fnErr = fn(ctx, c); fnErr != nil {
			return fnErr
		}
		return saveClosed(ctx, q, c)
	})
	switch {
	case fnErr != nil:
		return fnErr
	case errors.Is(err, pgx.ErrNoRows):
		return domain.ErrCountNotFound
	case err != nil:
		return fmt.Errorf("inventory: close stock count: %w", err)
	}
	return nil
}

// saveClosed stores the status of a closed count.
func saveClosed(ctx context.Context, q *sqlcgen.Queries, c *domain.StockCount) error {
	summary, approved := c.Summary()
	switch {
	case approved:
		return q.ApproveStockCount(ctx, sqlcgen.ApproveStockCountParams{
			ID: c.ID(), ClosedBy: c.ClosedBy(), ClosedAt: c.ClosedAt(),
			LinesTotal: int64(summary.Lines), LinesCounted: int64(summary.Counted), LinesExact: int64(summary.Exact),
			NetDifferenceValue: summary.NetValue, AbsoluteDifferenceValue: summary.AbsoluteValue,
		})
	case c.Status() == domain.CountCanceled:
		return q.CancelStockCount(ctx, sqlcgen.CancelStockCountParams{ID: c.ID(), ClosedBy: c.ClosedBy(), ClosedAt: c.ClosedAt()})
	}
	return fmt.Errorf("inventory: stock count %s left %s", c.ID(), c.Status())
}
