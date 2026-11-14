package domain

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
)

// Longest codes a line freezes: the catalog's SKU and unit code, and a
// UN/ECE Rec 20 code.
const (
	maxSKULength          = 64
	maxUnitDIANCodeLength = 3
)

// lineCodes is what the electronic document needs of a line, frozen at the
// time of the sale: the variant's SKU, the code of the product's base unit
// (the receipt shows it next to a quantity sold in it) and the DIAN code of
// the unit sold. Lines stored before they were frozen have them empty.
type lineCodes struct {
	sku, baseUnitCode, unitDIANCode string
}

// lineTax checks the codes and the IVA treatment of in, which must match
// its rate; a line without one (stored before it was frozen) gets the one
// its rate implies.
func lineTax(in LineInput) (lineCodes, vat.Treatment, error) {
	c := lineCodes{
		sku:          strings.TrimSpace(in.SKU),
		baseUnitCode: strings.TrimSpace(in.BaseUnitCode),
		unitDIANCode: strings.TrimSpace(in.UnitDIANCode),
	}
	if utf8.RuneCountInString(c.sku) > maxSKULength || utf8.RuneCountInString(c.baseUnitCode) > maxUnitCodeLength ||
		utf8.RuneCountInString(c.unitDIANCode) > maxUnitDIANCodeLength {
		return lineCodes{}, "", ErrInvalidLineCode
	}
	t, err := vat.Classify(in.TaxTreatment, in.TaxRate.IsZero())
	if errors.Is(err, vat.ErrInvalidTreatment) {
		return lineCodes{}, "", ErrInvalidTaxTreatment
	}
	return c, t, err
}

// SKU returns the variant's SKU at the time of the sale; empty for a line
// stored before it was frozen.
func (l SaleLine) SKU() string { return l.codes.sku }

// BaseUnitCode returns the code of the product's base unit at the time of
// the sale ("m"), which a quantity in the base unit is shown with.
func (l SaleLine) BaseUnitCode() string { return l.codes.baseUnitCode }

// UnitDIANCode returns the UN/ECE Rec 20 code of the unit sold ("MTR");
// empty if the unit had none.
func (l SaleLine) UnitDIANCode() string { return l.codes.unitDIANCode }

// TaxTreatment returns how the IVA treated the line: taxed, exempt or
// excluded.
func (l SaleLine) TaxTreatment() vat.Treatment { return l.taxTreatment }
