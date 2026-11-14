package domain

import (
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
)

// TaxCodeIVA is the DIAN's code of the IVA.
const TaxCodeIVA = "01"

// DefaultUnitCode is the UN/ECE Rec 20 code of "unit" (94), which a line
// carries when its unit has no DIAN code of its own.
const DefaultUnitCode = "94"

// LineTax is a tax a line of the document carries: its DIAN code, rate (as a
// fraction), taxable base and amount, in COP.
type LineTax struct {
	Code               string
	Rate, Base, Amount decimal.Decimal
}

// LineTaxes is the IVA a line carries in the document: none if excluded,
// at 0% if exempt and at its rate if taxed. base and amount are the sale
// line's, already rounded.
func LineTaxes(t vat.Treatment, rate, base, amount decimal.Decimal) []LineTax {
	if !t.Charged() {
		return nil
	}
	return []LineTax{{Code: TaxCodeIVA, Rate: rate, Base: base, Amount: amount}}
}

// UnitCode is the unit code a line carries: its unit's DIAN code, or
// DefaultUnitCode if it has none.
func UnitCode(dianCode string) string {
	if dianCode == "" {
		return DefaultUnitCode
	}
	return dianCode
}

// LineCode is the code a line carries: its variant's SKU, or, for a line
// stored before the SKU was frozen, the variant's id.
func LineCode(sku, variantID string) string {
	if sku == "" {
		return variantID
	}
	return sku
}
