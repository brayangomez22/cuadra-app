package domain

import "slices"

// TaxResponsibility is a tax responsibility of the customer's RUT that the
// electronic invoice reports (DIAN codes).
type TaxResponsibility string

// The responsibilities the electronic invoice asks of the buyer.
const (
	TaxLargeTaxpayer  TaxResponsibility = "O-13"    // gran contribuyente
	TaxSelfWithholder TaxResponsibility = "O-15"    // autorretenedor
	TaxVATWithholder  TaxResponsibility = "O-23"    // agente de retención de IVA
	TaxSimpleRegime   TaxResponsibility = "O-47"    // régimen simple de tributación
	TaxNotResponsible TaxResponsibility = "R-99-PN" // ninguna de las anteriores
)

// taxOrder lists the responsibilities in the order they are kept.
var taxOrder = []TaxResponsibility{
	TaxLargeTaxpayer, TaxSelfWithholder, TaxVATWithholder, TaxSimpleRegime, TaxNotResponsible,
}

// ParseTaxResponsibilities checks the codes in: known, not repeated, and
// R-99-PN alone. None means R-99-PN. The result is in a fixed order, so two
// equal sets compare equal.
func ParseTaxResponsibilities(in []string) ([]TaxResponsibility, error) {
	if len(in) == 0 {
		return []TaxResponsibility{TaxNotResponsible}, nil
	}
	seen := map[TaxResponsibility]bool{}
	for _, code := range in {
		r := TaxResponsibility(code)
		if !slices.Contains(taxOrder, r) || seen[r] {
			return nil, ErrInvalidTaxResponsibilities
		}
		seen[r] = true
	}
	if seen[TaxNotResponsible] && len(seen) > 1 {
		return nil, ErrInvalidTaxResponsibilities
	}
	out := make([]TaxResponsibility, 0, len(seen))
	for _, r := range taxOrder {
		if seen[r] {
			out = append(out, r)
		}
	}
	return out, nil
}
