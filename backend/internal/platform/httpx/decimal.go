package httpx

import "github.com/shopspring/decimal"

// FormatMoney writes an amount of the contract with 2 decimals, or up to 4
// when it has them (a unit price can): "10000.00", "8403.3613".
func FormatMoney(d decimal.Decimal) string {
	for places := int32(2); places < 4; places++ {
		if d.Equal(d.Truncate(places)) {
			return d.StringFixed(places)
		}
	}
	return d.StringFixed(4)
}

// FormatDecimal writes a quantity or a rate without trailing zeros: "2.5",
// "0.19", "-3".
func FormatDecimal(d decimal.Decimal) string { return d.String() }

// ParseDecimal reads a quantity or an amount the contract already checked; a
// value that still does not parse is reported as the domain error invalid.
func ParseDecimal(s string, invalid error) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Decimal{}, invalid
	}
	return d, nil
}
