package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
)

// MaxIdleDays is the longest an idle stock report looks back: two years.
const MaxIdleDays = 730

// Since is the first day a variant would have to be sold on not to be idle.
type Since struct {
	Days int
	Day  calendar.Date
}

// NewSince returns the day days before today, or ErrInvalidFilter if days
// is not between 1 and MaxIdleDays.
func NewSince(today calendar.Date, days int) (Since, error) {
	if days < 1 || days > MaxIdleDays {
		return Since{}, ErrInvalidFilter
	}
	return Since{Days: days, Day: today.AddDays(-days)}, nil
}

// IdleVariant is a variant with stock not sold since the report's day: its
// quantity in the base unit, its value at the average cost and its last
// sale (nil if it never sold).
type IdleVariant struct {
	VariantID  uuid.UUID
	Label, SKU string
	Quantity   decimal.Decimal
	Value      decimal.Decimal
	LastSoldAt *time.Time
}

// IdleProduct is a product's idle variants and what they add up to.
type IdleProduct struct {
	ProductID      uuid.UUID
	Name, UnitCode string
	Quantity       decimal.Decimal
	Value          decimal.Decimal
	LastSoldAt     *time.Time
	Variants       []IdleVariant
}

// IdleLine is a variant with idle stock, with its product.
type IdleLine struct {
	ProductID             uuid.UUID
	ProductName, UnitCode string
	VariantID             uuid.UUID
	Label, SKU            string
	Quantity, Value       decimal.Decimal
	LastSoldAt            *time.Time
}

// IdleStockReport is a page of the idle stock and the value of all of it.
type IdleStockReport struct {
	Since Since
	Rows  []IdleProduct
	Total int
	Value decimal.Decimal
	Costs bool
}

// HideCosts zeroes the report's values, for a caller who may not see costs.
func (r *IdleStockReport) HideCosts() {
	r.Costs = false
	r.Value = decimal.Zero
	for i := range r.Rows {
		r.Rows[i].Value = decimal.Zero
		for j := range r.Rows[i].Variants {
			r.Rows[i].Variants[j].Value = decimal.Zero
		}
	}
}

// AdjustmentRow is a group of stock adjustments: a user or a variant (Key)
// and its name (Label), with what came in and out in the base unit and its
// value at the cost of the moment (negative if lost).
type AdjustmentRow struct {
	Key, Label, UnitCode    string
	Adjustments             int
	QuantityIn, QuantityOut decimal.Decimal
	Value                   decimal.Decimal
}

// AdjustmentsReport is the period's stock adjustments by group.
type AdjustmentsReport struct {
	Period      Period
	Group       AdjustmentGroup
	Rows        []AdjustmentRow
	Adjustments int
	Value       decimal.Decimal
	Costs       bool
}

// Sum sets the report's totals from its rows.
func (r *AdjustmentsReport) Sum() {
	r.Adjustments, r.Value = 0, decimal.Zero
	for _, row := range r.Rows {
		r.Adjustments += row.Adjustments
		r.Value = r.Value.Add(row.Value)
	}
}

// HideCosts zeroes the report's values, for a caller who may not see costs.
func (r *AdjustmentsReport) HideCosts() {
	r.Costs = false
	r.Value = decimal.Zero
	for i := range r.Rows {
		r.Rows[i].Value = decimal.Zero
	}
}
