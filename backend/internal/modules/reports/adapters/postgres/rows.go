package postgres

import (
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// collect converts the rows a query returned, or passes its error on:
// collect(convert)(q.X(ctx, arg)).
func collect[R, T any](convert func(R) T) func([]R, error) ([]T, error) {
	return func(rows []R, err error) ([]T, error) {
		if err != nil {
			return nil, err
		}
		out := make([]T, 0, len(rows))
		for _, r := range rows {
			out = append(out, convert(r))
		}
		return out, nil
	}
}

// salesRow is a group of the sales report with its VAT breakdown.
func salesRow(key, label string, count int64, net, tax, total decimal.Decimal) domain.SalesRow {
	return domain.SalesRow{Key: key, Label: label, SalesFigures: domain.SalesFigures{
		Sales: int(count), Net: net, Tax: tax, Total: total, HasBreakdown: true,
	}}
}

// optional is nil for an empty filter.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
