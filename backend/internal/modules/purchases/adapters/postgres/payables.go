package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Payments returns the payments of a purchase of the tenant, the first one
// first.
func (r *PurchaseRepository) Payments(ctx context.Context, tenantID, purchaseID uuid.UUID) ([]*domain.SupplierPayment, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.SupplierPayment, error) {
		return sqlcgen.New(tx).SupplierPayments(ctx, purchaseID)
	})
	if err != nil {
		return nil, fmt.Errorf("purchases: supplier payments: %w", err)
	}
	out := make([]*domain.SupplierPayment, len(rows))
	for i, row := range rows {
		out[i], err = domain.RehydrateSupplierPayment(domain.SupplierPaymentSnapshot{
			ID: row.ID, TenantID: row.TenantID, PurchaseID: row.PurchaseID, SupplierID: row.SupplierID,
			Method: domain.PaymentMethod(row.Method), Amount: row.Amount, BalanceAfter: row.BalanceAfter,
			PaidOn: domain.DateOf(row.PaidOn), UserID: row.CreatedBy, CreatedAt: row.CreatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("purchases: supplier payment %s: %w", row.ID, err)
		}
	}
	return out, nil
}

// Payables returns a page of what the tenant owes each supplier and the
// totals, read in the same transaction.
func (r *PurchaseRepository) Payables(ctx context.Context, tenantID uuid.UUID, w domain.PayablesWindow, limit, offset int) (domain.PayablesPage, error) {
	rowLimit, rowOffset, err := db.PageBounds(limit, offset)
	if err != nil {
		return domain.PayablesPage{}, err
	}
	overdue, soon := w.OverdueBefore.Time(), w.DueSoonUntil.Time()
	var rows []sqlcgen.PayablesRow
	var totals sqlcgen.PayablesTotalsRow
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		rows, err = q.Payables(ctx, sqlcgen.PayablesParams{
			OverdueBefore: overdue, DueSoonUntil: soon, RowLimit: rowLimit, RowOffset: rowOffset,
		})
		if err != nil {
			return err
		}
		totals, err = q.PayablesTotals(ctx, sqlcgen.PayablesTotalsParams{OverdueBefore: overdue, DueSoonUntil: soon})
		return err
	})
	if err != nil {
		return domain.PayablesPage{}, fmt.Errorf("purchases: payables: %w", err)
	}
	page := domain.PayablesPage{Totals: domain.PayablesTotals{
		Balance: totals.Balance, Overdue: totals.Overdue, DueSoon: totals.DueSoon, Suppliers: int(totals.Suppliers),
	}}
	for _, row := range rows {
		page.Items = append(page.Items, domain.SupplierPayable{
			SupplierID: row.SupplierID, SupplierName: row.SupplierName, Balance: row.Balance, Overdue: row.Overdue,
			DueSoon: row.DueSoon, Purchases: int(row.Purchases), NextDue: domain.DateOf(row.NextDue),
		})
	}
	return page, nil
}
