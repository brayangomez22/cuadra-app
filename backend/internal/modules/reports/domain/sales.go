package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SalesFigures add up sales: how many, without VAT, the VAT and the total.
// A row by payment method has only the count and what was paid with it
// (HasBreakdown is false).
type SalesFigures struct {
	Sales        int
	Net, Tax     decimal.Decimal
	Total        decimal.Decimal
	HasBreakdown bool
}

// SalesRow is a group of the sales report: a day, a user, a payment
// method, a category or a location (Key) and its name (Label).
type SalesRow struct {
	Key, Label string
	SalesFigures
}

// SalesReport is the period's confirmed sales by group, and the totals of
// all of them.
type SalesReport struct {
	Period Period
	Group  SalesGroup
	Rows   []SalesRow
	Totals SalesFigures
}

// VariantSales is what a variant sold: in its product's base unit, without
// VAT and at its cost when it was sold.
type VariantSales struct {
	VariantID  uuid.UUID
	Label, SKU string
	Quantity   decimal.Decimal
	Profit
}

// ProductSales is what a product sold, with the detail of its variants.
type ProductSales struct {
	ProductID      uuid.UUID
	Name, UnitCode string
	Quantity       decimal.Decimal
	Profit
	Variants []VariantSales
}

// SoldVariant is what a variant sold in a period, with its product.
type SoldVariant struct {
	ProductID             uuid.UUID
	ProductName, UnitCode string
	VariantID             uuid.UUID
	Label, SKU            string
	Quantity              decimal.Decimal
	Profit
}

// ProductsReport is a page of the products sold in the period. Costs is
// false when the caller may not see costs: then every cost is zero.
type ProductsReport struct {
	Period Period
	Rows   []ProductSales
	Total  int
	Totals Profit
	Costs  bool
}

// HideCosts zeroes the report's costs, for a caller who may not see them.
func (r *ProductsReport) HideCosts() {
	r.Costs = false
	r.Totals.Cost = decimal.Zero
	for i := range r.Rows {
		r.Rows[i].Cost = decimal.Zero
		for j := range r.Rows[i].Variants {
			r.Rows[i].Variants[j].Cost = decimal.Zero
		}
	}
}

// SaleProfit is a confirmed sale's revenue and cost.
type SaleProfit struct {
	SaleID       uuid.UUID
	Number       int64
	LocationID   uuid.UUID
	LocationName string
	UserID       uuid.UUID
	UserName     string
	ConfirmedAt  time.Time
	Profit
}

// SaleProfitsReport is a page of the period's sales with their profit.
type SaleProfitsReport struct {
	Period Period
	Rows   []SaleProfit
	Total  int
	Totals Profit
}
