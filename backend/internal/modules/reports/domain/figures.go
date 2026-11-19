package domain

import "github.com/shopspring/decimal"

// moneyPlaces and ratePlaces are the decimals of an amount in pesos and of
// a rate (0.2500 is 25 %).
const (
	moneyPlaces = 2
	ratePlaces  = 4
)

// Money rounds an amount to pesos with cents, half-up.
func Money(d decimal.Decimal) decimal.Decimal { return d.Round(moneyPlaces) }

// Rate is part over whole rounded to 4 decimals; false when whole is zero.
func Rate(part, whole decimal.Decimal) (decimal.Decimal, bool) {
	if whole.IsZero() {
		return decimal.Zero, false
	}
	return part.DivRound(whole, ratePlaces), true
}

// Profit is what was sold without VAT and what it cost when it was sold.
type Profit struct {
	Revenue decimal.Decimal
	Cost    decimal.Decimal
}

// Amount is the revenue minus the cost, in pesos.
func (p Profit) Amount() decimal.Decimal { return Money(p.Revenue.Sub(p.Cost)) }

// Margin is the profit over the revenue; false without revenue.
func (p Profit) Margin() (decimal.Decimal, bool) { return Rate(p.Revenue.Sub(p.Cost), p.Revenue) }

// Add returns the sum of p and o.
func (p Profit) Add(o Profit) Profit {
	return Profit{Revenue: p.Revenue.Add(o.Revenue), Cost: p.Cost.Add(o.Cost)}
}
