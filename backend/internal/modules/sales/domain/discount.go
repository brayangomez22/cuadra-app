package domain

import "github.com/shopspring/decimal"

// DiscountLimit is the largest discount a user may give on a line, as a
// fraction of the line's gross amount (0.10 lets them discount 10%). The zero
// value allows no discount.
type DiscountLimit struct {
	max       decimal.Decimal
	unlimited bool
}

// NoDiscountLimit lets the user discount up to the whole line: it is the
// limit of whoever has the sales:discount permission.
var NoDiscountLimit = DiscountLimit{unlimited: true}

// NewDiscountLimit returns a limit of fraction, with 0 ≤ fraction ≤ 1 and at
// most four decimals.
func NewDiscountLimit(fraction decimal.Decimal) (DiscountLimit, error) {
	if fraction.IsNegative() || fraction.GreaterThan(decimal.NewFromInt(1)) || !fitsStorage(fraction) {
		return DiscountLimit{}, ErrInvalidDiscountLimit
	}
	return DiscountLimit{max: fraction}, nil
}

// Max returns the limit as a fraction of the line's gross amount; it is
// meaningless when Unlimited.
func (l DiscountLimit) Max() decimal.Decimal { return l.max }

// Unlimited reports whether there is no limit.
func (l DiscountLimit) Unlimited() bool { return l.unlimited }

// allows reports whether discount is within the limit for a line of gross.
// The comparison is exact: discount ≤ max × gross.
func (l DiscountLimit) allows(discount, gross decimal.Decimal) bool {
	return l.unlimited || discount.LessThanOrEqual(l.max.Mul(gross))
}
