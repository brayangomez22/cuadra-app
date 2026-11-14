package domain

import (
	"errors"

	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
)

// TaxRate is a product's IVA rate as a fraction (0.19 for 19%) and how the
// IVA treats it: taxed at a positive rate, or exempt or excluded at 0%, as
// the DIAN tells them apart. The zero value is a valid excluded 0% rate.
type TaxRate struct {
	rate      decimal.Decimal
	treatment vat.Treatment
}

// ParseTaxRate accepts 0 ≤ d < 1 with at most four decimals, with the
// treatment deduced from it (see NewTaxRate). Rates are not limited to
// today's (0, 5%, 19%) because they change by law.
func ParseTaxRate(d decimal.Decimal) (TaxRate, error) { return NewTaxRate(d, "") }

// NewTaxRate checks d (0 ≤ d < 1, four decimals) and that t matches it: a
// positive rate is taxed, a zero one exempt or excluded. Without t, a zero
// rate is excluded and a positive one taxed.
func NewTaxRate(d decimal.Decimal, t vat.Treatment) (TaxRate, error) {
	if d.IsNegative() || d.GreaterThanOrEqual(decimal.NewFromInt(1)) || !fitsStorage(d) {
		return TaxRate{}, ErrInvalidTaxRate
	}
	treatment, err := vat.Classify(t, d.IsZero())
	if errors.Is(err, vat.ErrInvalidTreatment) {
		return TaxRate{}, ErrInvalidTaxTreatment
	}
	return TaxRate{rate: d, treatment: treatment}, err
}

// Rate returns the rate as a fraction.
func (r TaxRate) Rate() decimal.Decimal { return r.rate }

// Treatment returns how the IVA treats the product.
func (r TaxRate) Treatment() vat.Treatment {
	if r.treatment == "" {
		return vat.Excluded
	}
	return r.treatment
}
