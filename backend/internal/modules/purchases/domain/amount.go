package domain

import "github.com/shopspring/decimal"

// Scales of the stored amounts: quantities, unit costs and rates keep four
// decimals (NUMERIC(18,4)); the amounts of a purchase are COP, rounded
// half-up to two.
const (
	amountScale = 4
	moneyScale  = 2
)

// maxAmount is the exclusive upper bound of NUMERIC(18,4): 14 integer digits.
var maxAmount = decimal.New(1, 14)

// fitsStorage reports whether d is stored exactly in NUMERIC(18,4).
func fitsStorage(d decimal.Decimal) bool {
	return d.Equal(d.Truncate(amountScale)) && d.Abs().LessThan(maxAmount)
}

// roundMoney rounds an amount of COP half-up to two decimals.
func roundMoney(d decimal.Decimal) decimal.Decimal { return d.Round(moneyScale) }
