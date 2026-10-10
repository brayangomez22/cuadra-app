package domain

import "github.com/shopspring/decimal"

// amountScale is the number of decimals stored for quantities and costs
// (NUMERIC(18,4) in Postgres).
const amountScale = 4

// maxAmount is the exclusive upper bound of NUMERIC(18,4): 14 integer digits.
var maxAmount = decimal.New(1, 14)

// fitsStorage reports whether d is stored exactly in NUMERIC(18,4): at most
// four decimals and below maxAmount in absolute value. Checking it here keeps
// the database from rounding or rejecting a value silently.
func fitsStorage(d decimal.Decimal) bool {
	return d.Equal(d.Truncate(amountScale)) && d.Abs().LessThan(maxAmount)
}

// validQuantity reports whether q is a positive quantity that fits storage.
func validQuantity(q decimal.Decimal) bool { return q.IsPositive() && fitsStorage(q) }

// validCost reports whether c is a non-negative cost that fits storage.
func validCost(c decimal.Decimal) bool { return !c.IsNegative() && fitsStorage(c) }
