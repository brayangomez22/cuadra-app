package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// SaleRepository implements domain.SaleRepository.
type SaleRepository struct{ db *db.DB }

// NewSaleRepository returns a SaleRepository on d.
func NewSaleRepository(d *db.DB) *SaleRepository { return &SaleRepository{db: d} }

// SaveConfirmed stores a confirmed sale in one transaction:
//
//  1. calls reserve with a context that carries the transaction, so the
//     inventory's stock exits join it;
//  2. draws the location's next number, which locks its sequence row until
//     commit (last, so the lock is held as briefly as possible);
//  3. inserts the sale, its lines and its payments.
//
// If any step fails, the transaction rolls back: no sale, no exits and no
// number used.
func (r *SaleRepository) SaveConfirmed(ctx context.Context, s *domain.Sale, reserve func(context.Context) error) error {
	// reserveErr keeps the inventory's error apart from the database errors,
	// which are translated and wrapped.
	var reserveErr error
	var number int64
	err := r.db.InTenantTx(ctx, s.TenantID(), func(ctx context.Context, tx pgx.Tx) error {
		if err := reserve(ctx); err != nil {
			reserveErr = err
			return err
		}
		q := sqlcgen.New(tx)
		var err error
		if number, err = q.NextSaleNumber(ctx, sqlcgen.NextSaleNumberParams{TenantID: s.TenantID(), LocationID: s.LocationID()}); err != nil {
			return err
		}
		return insertSale(ctx, q, s.Snapshot(), number)
	})
	switch {
	case reserveErr != nil:
		return reserveErr
	case violates(err, foreignKeyViolation, saleSequencesLocationFKey), violates(err, foreignKeyViolation, salesLocationFKey):
		return domain.ErrLocationNotFound
	case err != nil:
		return fmt.Errorf("sales: save sale %s: %w", s.ID(), err)
	}
	return s.AssignNumber(number)
}

func insertSale(ctx context.Context, q *sqlcgen.Queries, snap domain.SaleSnapshot, number int64) error {
	t := snap.Totals
	if err := q.InsertSale(ctx, sqlcgen.InsertSaleParams{
		ID: snap.ID, TenantID: snap.TenantID, LocationID: snap.LocationID, Number: number,
		UserID: snap.UserID, CustomerID: snap.CustomerID, CashSessionID: snap.CashSessionID, Status: string(snap.Status),
		Subtotal: t.Subtotal, Discounts: t.Discounts, Tax: t.Tax, Total: t.Total, Paid: t.Paid, Change: t.Change,
		CreatedAt: snap.CreatedAt, ConfirmedAt: snap.ConfirmedAt,
	}); err != nil {
		return err
	}
	if len(snap.Lines) > math.MaxInt32 || len(snap.Payments) > math.MaxInt32 {
		return errors.New("too many lines or payments")
	}
	for i, l := range snap.Lines {
		if err := q.InsertSaleLine(ctx, sqlcgen.InsertSaleLineParams{
			TenantID: snap.TenantID, SaleID: snap.ID, Position: int32(i),
			VariantID: l.VariantID, ProductID: l.ProductID, ProductName: l.ProductName, VariantLabel: nonEmpty(l.VariantLabel),
			Quantity: l.Quantity, UnitPrice: l.UnitPrice, PriceRule: string(l.PriceRule),
			TaxRate: l.TaxRate, Discount: l.Discount, Gross: l.Gross, Tax: l.Tax,
		}); err != nil {
			return err
		}
	}
	for i, p := range snap.Payments {
		if err := q.InsertSalePayment(ctx, sqlcgen.InsertSalePaymentParams{
			TenantID: snap.TenantID, SaleID: snap.ID, Position: int32(i),
			Method: string(p.Method), Amount: p.Amount,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Get returns a sale with its lines and payments, read in one transaction.
func (r *SaleRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Sale, error) {
	var sale *domain.Sale
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.GetSale(ctx, id)
		if err != nil {
			return err
		}
		sale, err = readSale(ctx, q, row)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrSaleNotFound
	case err != nil:
		return nil, fmt.Errorf("sales: get sale %s: %w", id, err)
	}
	return sale, nil
}

// Void locks the sale (SELECT ... FOR UPDATE), calls change with a context
// that carries the transaction (so the inventory's stock returns join it)
// and stores the void. A second void of the same sale waits for the first
// and then reads it already voided.
func (r *SaleRepository) Void(ctx context.Context, tenantID, id uuid.UUID, change func(context.Context, *domain.Sale) error) (*domain.Sale, error) {
	var changeErr error
	var sale *domain.Sale
	err := r.db.InTenantTx(ctx, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.LockSale(ctx, id)
		if err != nil {
			return err
		}
		if sale, err = readSale(ctx, q, row); err != nil {
			return err
		}
		if err := change(ctx, sale); err != nil {
			changeErr = err
			return err
		}
		v := sale.VoidInfo()
		if v == nil {
			return fmt.Errorf("sale %s was not voided", id)
		}
		affected, err := q.VoidSale(ctx, sqlcgen.VoidSaleParams{ID: id, VoidedAt: &v.At, VoidedBy: &v.UserID, VoidReason: &v.Reason})
		if err != nil {
			return err
		}
		if affected != 1 {
			return fmt.Errorf("sale %s is no longer confirmed", id)
		}
		return nil
	})
	switch {
	case changeErr != nil:
		return nil, changeErr
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrSaleNotFound
	case err != nil:
		return nil, fmt.Errorf("sales: void sale %s: %w", id, err)
	}
	return sale, nil
}

// readSale reads the lines and payments of a sale row and rebuilds it.
func readSale(ctx context.Context, q *sqlcgen.Queries, row sqlcgen.Sale) (*domain.Sale, error) {
	lines, err := q.ListSaleLines(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	payments, err := q.ListSalePayments(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	snap := domain.SaleSnapshot{
		ID: row.ID, TenantID: row.TenantID, LocationID: row.LocationID, UserID: row.UserID,
		CustomerID: row.CustomerID, CashSessionID: row.CashSessionID, Number: row.Number, Status: domain.SaleStatus(row.Status),
		Totals: domain.StoredTotals{
			Subtotal: row.Subtotal, Discounts: row.Discounts, Tax: row.Tax, Total: row.Total, Paid: row.Paid, Change: row.Change,
		},
		CreatedAt: row.CreatedAt, ConfirmedAt: row.ConfirmedAt,
	}
	if row.VoidedAt != nil && row.VoidedBy != nil && row.VoidReason != nil {
		snap.Void = &domain.VoidInfo{UserID: *row.VoidedBy, Reason: *row.VoidReason, At: *row.VoidedAt}
	}
	for _, l := range lines {
		snap.Lines = append(snap.Lines, domain.LineSnapshot{
			VariantID: l.VariantID, ProductID: l.ProductID, ProductName: l.ProductName, VariantLabel: valueOf(l.VariantLabel),
			Quantity: l.Quantity, UnitPrice: l.UnitPrice, PriceRule: domain.PriceRuleID(l.PriceRule),
			TaxRate: l.TaxRate, Discount: l.Discount, Gross: l.Gross, Tax: l.Tax,
		})
	}
	for _, p := range payments {
		snap.Payments = append(snap.Payments, domain.PaymentSnapshot{Method: domain.PaymentMethod(p.Method), Amount: p.Amount})
	}
	sale, err := domain.RehydrateSale(snap)
	if err != nil {
		return nil, fmt.Errorf("sale %s: %w", row.ID, err)
	}
	return sale, nil
}

// List returns one page of the sales confirmed in [From, To), newest first,
// and how many match in total, read in the same transaction.
func (r *SaleRepository) List(ctx context.Context, tenantID uuid.UUID, f domain.SaleFilter) (domain.SalePage, error) {
	limit, offset, err := pageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.SalePage{}, err
	}
	var rows []sqlcgen.ListSalesRow
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListSales(ctx, sqlcgen.ListSalesParams{
			FromTime: f.From, ToTime: f.To, LocationID: f.LocationID, RowLimit: limit, RowOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountSales(ctx, sqlcgen.CountSalesParams{FromTime: f.From, ToTime: f.To, LocationID: f.LocationID})
		return err
	})
	if err != nil {
		return domain.SalePage{}, fmt.Errorf("sales: list sales: %w", err)
	}
	page := domain.SalePage{Items: make([]domain.SaleSummary, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		sum := domain.SaleSummary{
			ID: row.ID, Number: row.Number, LocationID: row.LocationID, UserID: row.UserID,
			Status: domain.SaleStatus(row.Status), Total: row.Total, ConfirmedAt: row.ConfirmedAt.UTC(),
		}
		if row.VoidedAt != nil {
			at := row.VoidedAt.UTC()
			sum.VoidedAt = &at
		}
		page.Items = append(page.Items, sum)
	}
	return page, nil
}

// nonEmpty is s as a nullable column: nil when empty.
func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// valueOf is a nullable text column as a string: "" when NULL.
func valueOf(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
