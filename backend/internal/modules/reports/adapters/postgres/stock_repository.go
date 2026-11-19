package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// StockRepository implements domain.StockReader.
type StockRepository struct{ db *db.DB }

// NewStockRepository returns a StockRepository on d.
func NewStockRepository(d *db.DB) *StockRepository { return &StockRepository{db: d} }

// IdleVariants reads the variants with stock not sold since f.Since.
func (r *StockRepository) IdleVariants(ctx context.Context, f domain.IdleFilter) ([]domain.IdleLine, error) {
	arg := sqlcgen.IdleVariantsParams{
		LocationID: f.LocationID, CategoryID: f.CategoryID, Attribute: optional(f.Attribute),
		AttributeValue: optional(f.AttributeValue), SinceStart: f.Since.Day.Start(),
	}
	lines, err := db.Read(ctx, r.db, f.TenantID, func(tx pgx.Tx) ([]domain.IdleLine, error) {
		return collect(idleLine)(sqlcgen.New(tx).IdleVariants(ctx, arg))
	})
	if err != nil {
		return nil, fmt.Errorf("reports: read idle stock: %w", err)
	}
	return lines, nil
}

func idleLine(x sqlcgen.IdleVariantsRow) domain.IdleLine {
	l := domain.IdleLine{
		ProductID: x.ProductID, ProductName: x.ProductName, UnitCode: x.UnitCode, VariantID: x.VariantID,
		Label: x.Label, SKU: x.Sku, Quantity: x.Quantity, Value: x.Value,
	}
	// The query says "never sold" with the epoch: sqlc reads its max() as
	// never null.
	if x.LastSoldAt.Unix() != 0 {
		at := x.LastSoldAt.UTC()
		l.LastSoldAt = &at
	}
	return l
}

// Adjustments reads the period's stock adjustments by f's grouping.
func (r *StockRepository) Adjustments(ctx context.Context, f domain.AdjustmentFilter) (domain.AdjustmentsReport, error) {
	start, end := f.Period.Start(), f.Period.End()
	rows, err := db.Read(ctx, r.db, f.TenantID, func(tx pgx.Tx) ([]domain.AdjustmentRow, error) {
		q := sqlcgen.New(tx)
		if f.Group == domain.AdjustmentsByVariant {
			return collect(variantAdjustment)(q.AdjustmentsByVariant(ctx, sqlcgen.AdjustmentsByVariantParams{
				PeriodStart: start, PeriodEnd: end, LocationID: f.LocationID,
			}))
		}
		return collect(userAdjustment)(q.AdjustmentsByUser(ctx, sqlcgen.AdjustmentsByUserParams{
			PeriodStart: start, PeriodEnd: end, LocationID: f.LocationID,
		}))
	})
	if err != nil {
		return domain.AdjustmentsReport{}, fmt.Errorf("reports: read stock adjustments: %w", err)
	}
	return domain.AdjustmentsReport{Rows: rows}, nil
}

func userAdjustment(x sqlcgen.AdjustmentsByUserRow) domain.AdjustmentRow {
	return domain.AdjustmentRow{
		Key: x.Key, Label: x.Label, Adjustments: int(x.Adjustments),
		QuantityIn: x.QuantityIn, QuantityOut: x.QuantityOut, Value: x.Value,
	}
}

// variantAdjustment names a variant by its product and, if it has axes,
// their values: «Camiseta básica (Rojo · M)».
func variantAdjustment(x sqlcgen.AdjustmentsByVariantRow) domain.AdjustmentRow {
	label := x.ProductName
	if x.Label != "" {
		label += " (" + x.Label + ")"
	}
	return domain.AdjustmentRow{
		Key: x.Key, Label: label, UnitCode: x.UnitCode, Adjustments: int(x.Adjustments),
		QuantityIn: x.QuantityIn, QuantityOut: x.QuantityOut, Value: x.Value,
	}
}

// day parses a day the queries format as YYYY-MM-DD in Colombia.
func day(s string) (time.Time, error) { return time.Parse(time.DateOnly, s) }
