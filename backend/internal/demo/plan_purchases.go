package demo

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// The purchases: at most maxPurchaseLines lines per invoice (the contract
// allows 200; a supplier's invoice rarely has more than 100), received
// before opening, creditPurchases out of five on credit, and restocked when
// below a third (restockDivisor) of their level.
const (
	maxPurchaseLines = 100
	purchaseMinute   = openingHour*60 - 30
	creditPurchases  = 3
	restockDivisor   = 3
)

// initialStock buys every article for every location up to its level, one
// invoice per supplier (or more, by maxPurchaseLines).
func (h *history) initialStock(date time.Time) []Purchase {
	var out []Purchase
	for loc := range h.s.Locations {
		for p, product := range h.s.Products {
			for v := range product.Variants {
				k := stockKey{loc, p, v}
				h.level[k] = h.levelOf(product, loc)
			}
		}
		for sup := range h.s.Suppliers {
			out = append(out, h.purchases(date, loc, sup, func(stockKey) bool { return true })...)
		}
	}
	return out
}

// levelOf is how much of an article a location keeps: a few times what a
// sale line takes at most, less at the other locations.
func (h *history) levelOf(p Product, loc int) decimal.Decimal {
	times := int64(3 + h.rng.IntN(6))
	if loc > 0 {
		times = max(2, times*otherLocationShare/100)
	}
	return decimal.NewFromInt(int64(p.MaxQuantity) * times)
}

// restock is each supplier's weekly visit (on its weekday) to every
// location: what is below a third of its level is bought up to it.
func (h *history) restock(date time.Time) []Purchase {
	var out []Purchase
	for sup := range h.s.Suppliers {
		if int(date.Weekday()) != 1+sup%6 {
			continue
		}
		for loc := range h.s.Locations {
			low := func(k stockKey) bool {
				return h.stock[k].LessThan(h.level[k].Div(decimal.NewFromInt(restockDivisor)))
			}
			out = append(out, h.purchases(date, loc, sup, low)...)
		}
	}
	return out
}

// purchases are the invoices of sup at loc for its articles that need
// buying, up to their level, and adds them to the stock.
func (h *history) purchases(date time.Time, loc, sup int, needs func(stockKey) bool) []Purchase {
	var lines []Line
	for p, product := range h.s.Products {
		if product.Supplier != sup {
			continue
		}
		for v := range product.Variants {
			k := stockKey{loc, p, v}
			if q := h.level[k].Sub(h.stock[k]); needs(k) && q.IsPositive() {
				lines = append(lines, Line{Product: p, Variant: v, Quantity: q})
				h.stock[k] = h.level[k]
			}
		}
	}
	var out []Purchase
	for len(lines) > 0 {
		n := min(len(lines), maxPurchaseLines)
		h.invoices++
		out = append(out, Purchase{
			At: at(date, purchaseMinute+len(out)), Location: loc, Supplier: sup,
			Invoice: fmt.Sprintf("FE-%06d", h.invoices), Credit: h.rng.IntN(5) < creditPurchases,
			Lines: lines[:n],
		})
		lines = lines[n:]
	}
	return out
}
