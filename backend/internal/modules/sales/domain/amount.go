package domain

import "github.com/shopspring/decimal"

const (
	// amountScale is the number of decimals stored for quantities, prices
	// and rates (NUMERIC(18,4) in Postgres).
	amountScale = 4
	// moneyScale is the number of decimals of an amount of COP: line
	// amounts, discounts, taxes, totals and payments.
	moneyScale = 2
)

// maxAmount is the exclusive upper bound of NUMERIC(18,4): 14 integer digits.
var maxAmount = decimal.New(1, 14)

// fitsStorage reports whether d is stored exactly in NUMERIC(18,4): at most
// four decimals and below maxAmount in absolute value. Checking it here keeps
// the database from rounding or rejecting a value silently.
func fitsStorage(d decimal.Decimal) bool {
	return d.Equal(d.Truncate(amountScale)) && d.Abs().LessThan(maxAmount)
}

// isMoney reports whether d is an amount of COP that fits storage: at most
// two decimals.
func isMoney(d decimal.Decimal) bool {
	return d.Equal(d.Truncate(moneyScale)) && fitsStorage(d)
}

// roundMoney rounds d to two decimals, half-up (half away from zero, which is
// the same for the non-negative amounts of a sale).
func roundMoney(d decimal.Decimal) decimal.Decimal { return d.Round(moneyScale) }
