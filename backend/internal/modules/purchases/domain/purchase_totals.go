package domain

import (
	"slices"

	"github.com/shopspring/decimal"
)

// TaxLine is the VAT of a purchase at one rate: the base (the subtotal of
// its lines) and the VAT, the sum of theirs.
type TaxLine struct {
	Rate decimal.Decimal
	Base decimal.Decimal
	Tax  decimal.Decimal
}

// Totals are a purchase's amounts: the subtotal before VAT, the VAT by rate
// (lowest rate first) and in all, and the total.
type Totals struct {
	Subtotal decimal.Decimal
	Taxes    []TaxLine
	Tax      decimal.Decimal
	Total    decimal.Decimal
}

// computeTotals adds up the lines. Each line's VAT is rounded on its own,
// as the invoice shows it, so the VAT by rate is the sum of its lines'.
func computeTotals(lines []Line) Totals {
	t := Totals{Subtotal: decimal.Zero, Tax: decimal.Zero}
	for _, l := range lines {
		t.Subtotal = t.Subtotal.Add(l.subtotal)
		t.Tax = t.Tax.Add(l.tax)
		i := slices.IndexFunc(t.Taxes, func(x TaxLine) bool { return x.Rate.Equal(l.in.TaxRate) })
		if i < 0 {
			t.Taxes = append(t.Taxes, TaxLine{Rate: l.in.TaxRate, Base: decimal.Zero, Tax: decimal.Zero})
			i = len(t.Taxes) - 1
		}
		t.Taxes[i].Base = t.Taxes[i].Base.Add(l.subtotal)
		t.Taxes[i].Tax = t.Taxes[i].Tax.Add(l.tax)
	}
	slices.SortFunc(t.Taxes, func(x, y TaxLine) int { return x.Rate.Cmp(y.Rate) })
	t.Total = t.Subtotal.Add(t.Tax)
	return t
}
