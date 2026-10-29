package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Search returns one page of the tenant's purchases matching f, the latest
// invoice first, and how many match in total, read in the same transaction.
func (r *PurchaseRepository) Search(ctx context.Context, f domain.PurchaseFilter) ([]domain.PurchaseSummary, int, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	params := sqlcgen.SearchPurchasesParams{
		SupplierID: f.SupplierID, Open: openFilter(f.Status), IssuedFrom: dayTime(f.From), IssuedTo: dayTime(f.To),
		RowLimit: limit, RowOffset: offset,
	}
	var rows []sqlcgen.SearchPurchasesRow
	var total int64
	err = r.db.WithTenantTx(ctx, f.TenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.SearchPurchases(ctx, params); err != nil {
			return err
		}
		total, err = q.CountPurchases(ctx, sqlcgen.CountPurchasesParams{
			SupplierID: params.SupplierID, Open: params.Open, IssuedFrom: params.IssuedFrom, IssuedTo: params.IssuedTo,
		})
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("purchases: search purchases: %w", err)
	}
	out := make([]domain.PurchaseSummary, len(rows))
	for i, row := range rows {
		out[i] = domain.PurchaseSummary{
			ID: row.ID, SupplierID: row.SupplierID, SupplierName: row.SupplierName, LocationID: row.LocationID,
			InvoiceNumber: row.InvoiceNumber, IssuedOn: domain.DateOf(row.IssuedOn), Payment: domain.Payment(row.Payment),
			Total: row.Total, BalanceDue: row.BalanceDue, Lines: int(row.LineCount), CreatedAt: row.CreatedAt,
		}
		if row.DueOn != nil {
			out[i].DueOn = new(domain.DateOf(*row.DueOn))
		}
	}
	return out, int(total), nil
}

// BalancesDue returns what the tenant owes each supplier among ids that it
// owes something.
func (r *PurchaseRepository) BalancesDue(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.SupplierBalancesRow, error) {
		return sqlcgen.New(tx).SupplierBalances(ctx, ids)
	})
	if err != nil {
		return nil, fmt.Errorf("purchases: supplier balances: %w", err)
	}
	out := make(map[uuid.UUID]decimal.Decimal, len(rows))
	for _, row := range rows {
		out[row.SupplierID] = row.BalanceDue
	}
	return out, nil
}

// openFilter is whether a status asks for purchases with a balance; nil for
// every status.
func openFilter(s domain.PurchaseStatus) *bool {
	switch s {
	case domain.PurchaseOpen:
		return new(true)
	case domain.PurchasePaid:
		return new(false)
	default:
		return nil
	}
}

func dayTime(d *domain.Date) *time.Time {
	if d == nil {
		return nil
	}
	return new(d.Time())
}
