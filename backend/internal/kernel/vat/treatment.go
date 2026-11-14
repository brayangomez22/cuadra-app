// Package vat is how the IVA treats what is sold, as the DIAN tells them
// apart: taxed at a rate, exempt (taxed at 0%: it goes in the document with
// its IVA at 0) or excluded (outside the IVA: no IVA in the document). It is
// shared by the catalog, the sales and the electronic invoice.
package vat

import "errors"

// Treatment is how the IVA treats a product.
type Treatment string

// The treatments of the IVA.
const (
	Taxed    Treatment = "taxed"
	Exempt   Treatment = "exempt"
	Excluded Treatment = "excluded"
)

// ErrInvalidTreatment is a treatment that does not exist or does not match
// the rate.
var ErrInvalidTreatment = errors.New("vat: invalid treatment")

// Classify checks t against whether the rate is zero. A positive rate is
// always taxed; a zero rate is exempt or excluded. Without t, it is deduced:
// taxed with a positive rate, excluded (the most common) with a zero one.
func Classify(t Treatment, zeroRate bool) (Treatment, error) {
	switch {
	case t == "" && zeroRate:
		return Excluded, nil
	case t == "":
		return Taxed, nil
	case t == Taxed && !zeroRate, (t == Exempt || t == Excluded) && zeroRate:
		return t, nil
	}
	return "", ErrInvalidTreatment
}

// Charged is whether the document carries the IVA of a line so treated
// (at its rate, which is 0 if exempt). An excluded line carries none.
func (t Treatment) Charged() bool { return t != Excluded }
