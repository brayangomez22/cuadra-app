package domain

import (
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

// maxUnitCodeLength is the longest unit code a tenant can define.
const maxUnitCodeLength = 16

// LineUnit is the unit a line is sold in. The zero value and BaseUnit are
// the product's base unit; another unit has its code, frozen at the time of
// the sale, and how many base units one of it holds.
type LineUnit struct {
	// Code is the alternate unit's code ("caja"); empty for the base unit.
	Code string
	// Factor is how many base units one unit of the line holds: 1 for the
	// base unit, 12 for a box of 12.
	Factor decimal.Decimal
}

// BaseUnit is the product's base unit, in which the stock is kept.
var BaseUnit = LineUnit{Factor: decimal.NewFromInt(1)}

// newLineUnit validates u: the base unit has factor 1 (or none), and an
// alternate unit a code and a positive factor that fits storage.
func newLineUnit(u LineUnit) (LineUnit, error) {
	code := strings.TrimSpace(u.Code)
	if code == "" {
		if !u.Factor.IsZero() && !u.Factor.Equal(BaseUnit.Factor) {
			return LineUnit{}, ErrInvalidUnitFactor
		}
		return BaseUnit, nil
	}
	if utf8.RuneCountInString(code) > maxUnitCodeLength {
		return LineUnit{}, ErrInvalidUnitCode
	}
	if !u.Factor.IsPositive() || !fitsStorage(u.Factor) {
		return LineUnit{}, ErrInvalidUnitFactor
	}
	return LineUnit{Code: code, Factor: u.Factor}, nil
}

// IsBase reports whether u is the product's base unit.
func (u LineUnit) IsBase() bool { return u.Code == "" }

// Unit returns the unit the line was sold in.
func (l SaleLine) Unit() LineUnit { return l.unit }

// BaseQuantity returns the quantity in the product's base unit: what the
// stock counts. It is Quantity × the unit's factor.
func (l SaleLine) BaseQuantity() decimal.Decimal { return l.quantity.Mul(l.unit.Factor) }
