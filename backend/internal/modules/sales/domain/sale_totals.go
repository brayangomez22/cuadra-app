package domain

import (
	"slices"

	"github.com/shopspring/decimal"
)

// TaxBreakdown is the IVA of the lines at one rate: the sum of their bases
// and of their taxes.
type TaxBreakdown struct {
	Rate decimal.Decimal
	Base decimal.Decimal
	Tax  decimal.Decimal
}

// Totals are a sale's amounts. Subtotal is the sum of the lines' gross
// amounts; Total = Subtotal − Discounts + Tax. Taxes has one entry per IVA
// rate, by rate.
type Totals struct {
	Subtotal  decimal.Decimal
	Discounts decimal.Decimal
	Tax       decimal.Decimal
	Total     decimal.Decimal
	Taxes     []TaxBreakdown
}

// Totals adds up the lines. The taxes by rate are the sums of the lines'
// already rounded amounts, so nothing is rounded here.
func (s *Sale) Totals() Totals {
	t := Totals{Subtotal: decimal.Zero, Discounts: decimal.Zero, Tax: decimal.Zero}
	byRate := map[string]int{}
	for _, l := range s.lines {
		t.Subtotal = t.Subtotal.Add(l.gross)
		t.Discounts = t.Discounts.Add(l.discount)
		t.Tax = t.Tax.Add(l.tax)

		// The key normalizes 0.19 and 0.1900 to the same rate.
		key := l.taxRate.String()
		i, ok := byRate[key]
		if !ok {
			i = len(t.Taxes)
			byRate[key] = i
			t.Taxes = append(t.Taxes, TaxBreakdown{Rate: l.taxRate, Base: decimal.Zero, Tax: decimal.Zero})
		}
		t.Taxes[i].Base = t.Taxes[i].Base.Add(l.Base())
		t.Taxes[i].Tax = t.Taxes[i].Tax.Add(l.tax)
	}
	slices.SortFunc(t.Taxes, func(a, b TaxBreakdown) int { return a.Rate.Cmp(b.Rate) })
	t.Total = t.Subtotal.Sub(t.Discounts).Add(t.Tax)
	return t
}

// Paid returns the sum of the payments.
func (s *Sale) Paid() decimal.Decimal {
	paid := decimal.Zero
	for _, p := range s.payments {
		paid = paid.Add(p.amount)
	}
	return paid
}

// Change returns the cash given back; zero for a draft.
func (s *Sale) Change() decimal.Decimal { return s.change }

// CreditAmount returns the part of the sale paid on credit, zero if none.
func (s *Sale) CreditAmount() decimal.Decimal {
	for _, p := range s.payments {
		if p.method == PaymentCredit {
			return p.amount
		}
	}
	return decimal.Zero
}
