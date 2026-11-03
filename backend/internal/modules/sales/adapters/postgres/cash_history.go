package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Search returns one page of the tenant's sessions matching f, the latest
// opened first, and how many match in total, read in the same transaction.
func (r *CashSessionRepository) Search(ctx context.Context, f domain.CashSessionFilter) ([]domain.CashSessionSummary, int, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	params := sqlcgen.SearchCashSessionsParams{
		LocationID: f.LocationID, OpenedFrom: f.OpenedFrom, OpenedBefore: f.OpenedBefore,
		RowLimit: limit, RowOffset: offset,
	}
	if f.Status != "" {
		params.Status = new(string(f.Status))
	}
	var rows []sqlcgen.SearchCashSessionsRow
	var total int64
	err = r.db.WithTenantTx(ctx, f.TenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.SearchCashSessions(ctx, params); err != nil {
			return err
		}
		total, err = q.CountCashSessions(ctx, sqlcgen.CountCashSessionsParams{
			LocationID: params.LocationID, Status: params.Status, OpenedFrom: params.OpenedFrom,
			OpenedBefore: params.OpenedBefore,
		})
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("sales: search cash sessions: %w", err)
	}
	out := make([]domain.CashSessionSummary, len(rows))
	for i, row := range rows {
		out[i] = domain.CashSessionSummary{
			ID: row.ID, LocationID: row.LocationID, UserID: row.UserID, Status: domain.CashSessionStatus(row.Status),
			OpeningFloat: row.OpeningFloat, OpenedAt: row.OpenedAt.UTC(), ClosedBy: row.ClosedBy,
			Difference: row.Difference, SalesTotal: row.SalesTotal,
		}
		if row.ClosedAt != nil {
			out[i].ClosedAt = new(row.ClosedAt.UTC())
		}
		if row.SalesCount != nil {
			out[i].SalesCount = int(*row.SalesCount)
		}
	}
	return out, int(total), nil
}
