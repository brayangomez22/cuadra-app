package domain

import "github.com/shopspring/decimal"

// moneyScale is the decimals of an amount of COP.
const moneyScale = 2

// isMoney reports whether d is an amount of COP: at most two decimals.
func isMoney(d decimal.Decimal) bool {
	return d.Equal(d.Truncate(moneyScale))
}
