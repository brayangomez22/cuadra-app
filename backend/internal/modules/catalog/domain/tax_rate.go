package domain

import "github.com/shopspring/decimal"

// TaxRate is a product's IVA rate as a fraction (0.19 for 19%). The zero
// value is a valid 0% rate. The exempt / excluded distinction the DIAN needs
// is decided with electronic invoicing (T39).
type TaxRate struct {
	rate decimal.Decimal
}

// ParseTaxRate accepts 0 ≤ d < 1 with at most four decimals. Rates are not
// limited to today's (0, 5%, 19%) because they change by law.
func ParseTaxRate(d decimal.Decimal) (TaxRate, error) {
	if d.IsNegative() || d.GreaterThanOrEqual(decimal.NewFromInt(1)) || !fitsStorage(d) {
		return TaxRate{}, ErrInvalidTaxRate
	}
	return TaxRate{rate: d}, nil
}

// Rate returns the rate as a fraction.
func (r TaxRate) Rate() decimal.Decimal { return r.rate }
