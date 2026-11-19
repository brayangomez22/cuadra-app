package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// NewProductsReport groups the sold variants by product, orders the
// products by sort (most first, then by name) and their variants by
// revenue, and returns the page with the count and the totals of every
// product.
func NewProductsReport(lines []SoldVariant, sort ProductSort, page Page) ProductsReport {
	var r ProductsReport
	index := map[uuid.UUID]int{}
	for _, l := range lines {
		i, ok := index[l.ProductID]
		if !ok {
			i = len(r.Rows)
			index[l.ProductID] = i
			r.Rows = append(r.Rows, ProductSales{ProductID: l.ProductID, Name: l.ProductName, UnitCode: l.UnitCode})
		}
		p := &r.Rows[i]
		p.Quantity = p.Quantity.Add(l.Quantity)
		p.Profit = p.Add(l.Profit)
		p.Variants = append(p.Variants, VariantSales{VariantID: l.VariantID, Label: l.Label, SKU: l.SKU, Quantity: l.Quantity, Profit: l.Profit})
		r.Totals = r.Totals.Add(l.Profit)
	}
	key := productKey(sort)
	slices.SortStableFunc(r.Rows, func(a, b ProductSales) int { return mostFirst(key(a), key(b), a.Name, b.Name) })
	for i := range r.Rows {
		slices.SortStableFunc(r.Rows[i].Variants, func(a, b VariantSales) int {
			return mostFirst(a.Revenue, b.Revenue, a.Label, b.Label)
		})
	}
	r.Total = len(r.Rows)
	r.Rows = pageOf(r.Rows, page)
	return r
}

// productKey is the figure sort orders the products by.
func productKey(sort ProductSort) func(ProductSales) decimal.Decimal {
	switch sort {
	case ByQuantity:
		return func(p ProductSales) decimal.Decimal { return p.Quantity }
	case ByProfit:
		return func(p ProductSales) decimal.Decimal { return p.Amount() }
	default:
		return func(p ProductSales) decimal.Decimal { return p.Revenue }
	}
}

// NewIdleStockReport groups the idle variants by product, orders the
// products by value (by quantity if not byValue; most first, then by name)
// and their variants the same way, and returns the page with the count and
// the value of all of it. A product's last sale is its variants' latest.
func NewIdleStockReport(lines []IdleLine, byValue bool, page Page) IdleStockReport {
	var r IdleStockReport
	index := map[uuid.UUID]int{}
	for _, l := range lines {
		i, ok := index[l.ProductID]
		if !ok {
			i = len(r.Rows)
			index[l.ProductID] = i
			r.Rows = append(r.Rows, IdleProduct{ProductID: l.ProductID, Name: l.ProductName, UnitCode: l.UnitCode})
		}
		p := &r.Rows[i]
		p.Quantity = p.Quantity.Add(l.Quantity)
		p.Value = p.Value.Add(l.Value)
		p.LastSoldAt = latest(p.LastSoldAt, l.LastSoldAt)
		p.Variants = append(p.Variants, IdleVariant{VariantID: l.VariantID, Label: l.Label, SKU: l.SKU,
			Quantity: l.Quantity, Value: l.Value, LastSoldAt: l.LastSoldAt})
		r.Value = r.Value.Add(l.Value)
	}
	weight := func(q, v decimal.Decimal) decimal.Decimal {
		if byValue {
			return v
		}
		return q
	}
	slices.SortStableFunc(r.Rows, func(a, b IdleProduct) int {
		return mostFirst(weight(a.Quantity, a.Value), weight(b.Quantity, b.Value), a.Name, b.Name)
	})
	for i := range r.Rows {
		slices.SortStableFunc(r.Rows[i].Variants, func(a, b IdleVariant) int {
			return mostFirst(weight(a.Quantity, a.Value), weight(b.Quantity, b.Value), a.Label, b.Label)
		})
	}
	r.Total = len(r.Rows)
	r.Rows = pageOf(r.Rows, page)
	return r
}

// mostFirst orders by a figure, greatest first, then by name.
func mostFirst(a, b decimal.Decimal, nameA, nameB string) int {
	if c := b.Cmp(a); c != 0 {
		return c
	}
	return strings.Compare(nameA, nameB)
}

// latest returns the later of two instants, either of which may be nil.
func latest(a, b *time.Time) *time.Time {
	if a == nil || (b != nil && b.After(*a)) {
		return b
	}
	return a
}

// pageOf returns the page of rows.
func pageOf[T any](rows []T, p Page) []T {
	from := min(p.Offset, len(rows))
	return rows[from:min(from+p.Limit, len(rows))]
}
