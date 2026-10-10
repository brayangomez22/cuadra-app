package httpadapter

import (
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

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

// formatRate writes a tax rate without trailing zeros: "0.19", "0".
func formatRate(r domain.TaxRate) string { return r.Rate().String() }

// parseAmount reads an amount the contract already checked; a value that
// still does not parse is reported as the domain error invalid.
func parseAmount(s string, invalid error) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Decimal{}, invalid
	}
	return d, nil
}
