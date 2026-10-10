package httpadapter

import "github.com/shopspring/decimal"

// formatMoney writes an amount with 2 decimals, or up to 4 when it has them:
// "10000.00", "8333.3333".
func formatMoney(d decimal.Decimal) string {
	for places := int32(2); places < 4; places++ {
		if d.Equal(d.Truncate(places)) {
			return d.StringFixed(places)
		}
	}
	return d.StringFixed(4)
}

// formatQuantity writes a quantity without trailing zeros: "2.5", "-3", "10".
func formatQuantity(d decimal.Decimal) string { return d.String() }

// parseDecimal reads a quantity or an amount the contract already checked; a
// value that still does not parse is reported as the domain error invalid.
func parseDecimal(s string, invalid error) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Decimal{}, invalid
	}
	return d, nil
}
