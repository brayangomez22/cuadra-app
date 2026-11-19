// Package postgres reads the business reports from PostgreSQL. It owns no
// table: it reads those of sales, inventory, catalog, audit and identity,
// always inside db.WithTenantTx, so Row-Level Security scopes every read
// to the tenant. Grouping, ordering and totals beyond SQL are the domain's.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// SalesRepository implements domain.SalesReader.
type SalesRepository struct{ db *db.DB }

// NewSalesRepository returns a SalesRepository on d.
func NewSalesRepository(d *db.DB) *SalesRepository { return &SalesRepository{db: d} }

// salesRange is the period and location every sales query filters by.
type salesRange = sqlcgen.SalesTotalsParams

func rangeOf(s domain.Scope, p domain.Period) salesRange {
	return salesRange{PeriodStart: p.Start(), PeriodEnd: p.End(), LocationID: s.LocationID}
}

// Sales reads the rows of f's grouping and the totals of every sale.
func (r *SalesRepository) Sales(ctx context.Context, f domain.SalesFilter) (domain.SalesReport, error) {
	arg := rangeOf(f.Scope, f.Period)
	report, err := db.Read(ctx, r.db, f.TenantID, func(tx pgx.Tx) (domain.SalesReport, error) {
		q := sqlcgen.New(tx)
		t, err := q.SalesTotals(ctx, arg)
		if err != nil {
			return domain.SalesReport{}, err
		}
		rows, err := salesRows(ctx, q, f.Group, arg)
		if err != nil {
			return domain.SalesReport{}, err
		}
		return domain.SalesReport{Rows: rows, Totals: domain.SalesFigures{
			Sales: int(t.SalesCount), Net: t.Net, Tax: t.Tax, Total: t.Total, HasBreakdown: true,
		}}, nil
	})
	if err != nil {
		return domain.SalesReport{}, fmt.Errorf("reports: read sales: %w", err)
	}
	return report, nil
}

// salesRows reads the groups of the sales report.
func salesRows(ctx context.Context, q *sqlcgen.Queries, group domain.SalesGroup, arg salesRange) ([]domain.SalesRow, error) {
	switch group {
	case domain.SalesByUser:
		return collect(func(x sqlcgen.SalesByUserRow) domain.SalesRow {
			return salesRow(x.Key, x.Label, x.SalesCount, x.Net, x.Tax, x.Total)
		})(q.SalesByUser(ctx, sqlcgen.SalesByUserParams(arg)))
	case domain.SalesByLocation:
		return collect(func(x sqlcgen.SalesByLocationRow) domain.SalesRow {
			return salesRow(x.Key, x.Label, x.SalesCount, x.Net, x.Tax, x.Total)
		})(q.SalesByLocation(ctx, sqlcgen.SalesByLocationParams(arg)))
	case domain.SalesByCategory:
		return collect(func(x sqlcgen.SalesByCategoryRow) domain.SalesRow {
			return salesRow(x.Key, x.Label, x.SalesCount, x.Net, x.Tax, x.Total)
		})(q.SalesByCategory(ctx, sqlcgen.SalesByCategoryParams(arg)))
	case domain.SalesByPaymentMethod:
		return collect(func(x sqlcgen.SalesByPaymentMethodRow) domain.SalesRow {
			return domain.SalesRow{Key: x.Key, Label: x.Key, SalesFigures: domain.SalesFigures{Sales: int(x.SalesCount), Total: x.Total}}
		})(q.SalesByPaymentMethod(ctx, sqlcgen.SalesByPaymentMethodParams(arg)))
	default:
		return collect(func(x sqlcgen.SalesByDayRow) domain.SalesRow {
			return salesRow(x.Key, x.Key, x.SalesCount, x.Net, x.Tax, x.Total)
		})(q.SalesByDay(ctx, sqlcgen.SalesByDayParams(arg)))
	}
}

// SoldVariants reads what each variant sold in the period.
func (r *SalesRepository) SoldVariants(ctx context.Context, f domain.ProductFilter) ([]domain.SoldVariant, error) {
	arg := sqlcgen.SoldVariantsParams{
		PeriodStart: f.Period.Start(), PeriodEnd: f.Period.End(), LocationID: f.LocationID,
		CategoryID: f.CategoryID, Attribute: optional(f.Attribute), AttributeValue: optional(f.AttributeValue),
	}
	lines, err := db.Read(ctx, r.db, f.TenantID, func(tx pgx.Tx) ([]domain.SoldVariant, error) {
		return collect(func(x sqlcgen.SoldVariantsRow) domain.SoldVariant {
			return domain.SoldVariant{
				ProductID: x.ProductID, ProductName: x.ProductName, UnitCode: x.UnitCode,
				VariantID: x.VariantID, Label: x.Label, SKU: x.Sku, Quantity: x.Quantity,
				Profit: domain.Profit{Revenue: x.Revenue, Cost: x.Cost},
			}
		})(sqlcgen.New(tx).SoldVariants(ctx, arg))
	})
	if err != nil {
		return nil, fmt.Errorf("reports: read sold variants: %w", err)
	}
	return lines, nil
}

// SaleProfits reads a page of the period's sales with their profit and the
// totals of all of them.
func (r *SalesRepository) SaleProfits(ctx context.Context, f domain.SaleProfitFilter) (domain.SaleProfitsReport, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.SaleProfitsReport{}, err
	}
	arg := rangeOf(f.Scope, f.Period)
	report, err := db.Read(ctx, r.db, f.TenantID, func(tx pgx.Tx) (domain.SaleProfitsReport, error) {
		q := sqlcgen.New(tx)
		t, err := q.SaleProfitsTotals(ctx, sqlcgen.SaleProfitsTotalsParams(arg))
		if err != nil {
			return domain.SaleProfitsReport{}, err
		}
		rows, err := collect(saleProfit)(q.SaleProfitsPage(ctx, sqlcgen.SaleProfitsPageParams{
			PeriodStart: arg.PeriodStart, PeriodEnd: arg.PeriodEnd, LocationID: arg.LocationID, RowLimit: limit, RowOffset: offset,
		}))
		if err != nil {
			return domain.SaleProfitsReport{}, err
		}
		return domain.SaleProfitsReport{Rows: rows, Total: int(t.SalesCount), Totals: domain.Profit{Revenue: t.Revenue, Cost: t.Cost}}, nil
	})
	if err != nil {
		return domain.SaleProfitsReport{}, fmt.Errorf("reports: read sale profits: %w", err)
	}
	return report, nil
}

func saleProfit(x sqlcgen.SaleProfitsPageRow) domain.SaleProfit {
	return domain.SaleProfit{
		SaleID: x.SaleID, Number: x.Number, LocationID: x.LocationID, LocationName: x.LocationName,
		UserID: x.UserID, UserName: x.UserName, ConfirmedAt: x.ConfirmedAt,
		Profit: domain.Profit{Revenue: x.Revenue, Cost: x.Cost},
	}
}
