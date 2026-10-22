package domain

import "github.com/shopspring/decimal"

// ToBaseUnit converts a quantity and a unit cost in an alternate unit,
// where one unit holds factor base units, to the product's base unit, in
// which the stock is kept: qty × factor at cost ÷ factor. The quantity must
// fit four decimals exactly (ErrQuantityPrecision): the stock does not lose
// precision. The cost is rounded half-up to four decimals; it only feeds
// the average cost.
func ToBaseUnit(qty, unitCost, factor decimal.Decimal) (baseQty, baseCost decimal.Decimal, err error) {
	if !factor.IsPositive() || !fitsStorage(factor) {
		return decimal.Decimal{}, decimal.Decimal{}, ErrInvalidUnitFactor
	}
	baseQty = qty.Mul(factor)
	if !fitsStorage(baseQty) {
		return decimal.Decimal{}, decimal.Decimal{}, ErrQuantityPrecision
	}
	return baseQty, unitCost.DivRound(factor, amountScale), nil
}
