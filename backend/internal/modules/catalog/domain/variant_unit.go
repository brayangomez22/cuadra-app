package domain

import (
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// maxVariantUnits caps the alternate units of one variant.
const maxVariantUnits = 10

// VariantUnitParams holds the data of an alternate unit of a variant.
type VariantUnitParams struct {
	// Unit is a unit of the tenant other than the product's base unit.
	Unit UnitRef
	// Factor is how many base units one alternate unit holds: 12 for a box
	// of 12 units, 0.001 for a gram of something counted in kilograms.
	Factor decimal.Decimal
	// Price is the price of one alternate unit, without IVA.
	Price decimal.Decimal
	// Barcode is optional.
	Barcode string
}

// VariantUnit is another unit in which a variant is sold or bought, with
// its conversion to the product's base unit and its own price. The stock is
// always kept in the base unit.
type VariantUnit struct {
	unit    UnitRef
	factor  decimal.Decimal
	price   decimal.Decimal
	barcode string
}

// newVariantUnit checks an alternate unit's own invariants; the ones
// involving the product's base unit and the other units are the product's.
func newVariantUnit(p VariantUnitParams) (VariantUnit, error) {
	if p.Unit.ID == uuid.Nil {
		return VariantUnit{}, ErrInvalidUnit
	}
	if !p.Factor.IsPositive() || !fitsStorage(p.Factor) {
		return VariantUnit{}, ErrInvalidUnitFactor
	}
	if !validPrice(p.Price) {
		return VariantUnit{}, ErrInvalidPrice
	}
	barcode, err := normalizeBarcode(p.Barcode)
	if err != nil {
		return VariantUnit{}, err
	}
	return VariantUnit{unit: p.Unit, factor: p.Factor, price: p.Price, barcode: barcode}, nil
}

// Unit returns the alternate unit.
func (u VariantUnit) Unit() UnitRef { return u.unit }

// Factor returns how many base units one alternate unit holds.
func (u VariantUnit) Factor() decimal.Decimal { return u.factor }

// Price returns the price of one alternate unit, without IVA.
func (u VariantUnit) Price() decimal.Decimal { return u.price }

// Barcode returns the barcode, or "" if the unit has none.
func (u VariantUnit) Barcode() string { return u.barcode }

func (u VariantUnit) params() VariantUnitParams {
	return VariantUnitParams{Unit: u.unit, Factor: u.factor, Price: u.price, Barcode: u.barcode}
}

// Units returns the variant's alternate units, from the smallest factor to
// the largest.
func (v *ProductVariant) Units() []VariantUnit { return slices.Clone(v.units) }

// Unit returns the alternate unit unitID of the variant, or
// ErrVariantUnitNotFound.
func (v *ProductVariant) Unit(unitID uuid.UUID) (VariantUnit, error) {
	i := slices.IndexFunc(v.units, func(u VariantUnit) bool { return u.unit.ID == unitID })
	if i < 0 {
		return VariantUnit{}, ErrVariantUnitNotFound
	}
	return v.units[i], nil
}

// newVariantUnits builds a variant's alternate units sorted by factor,
// without the same unit twice.
func newVariantUnits(params []VariantUnitParams) ([]VariantUnit, error) {
	if len(params) > maxVariantUnits {
		return nil, ErrTooManyVariantUnits
	}
	units := make([]VariantUnit, 0, len(params))
	for _, p := range params {
		u, err := newVariantUnit(p)
		if err != nil {
			return nil, err
		}
		if slices.ContainsFunc(units, func(other VariantUnit) bool { return other.unit.ID == u.unit.ID }) {
			return nil, ErrDuplicateVariantUnit
		}
		units = append(units, u)
	}
	slices.SortStableFunc(units, func(a, b VariantUnit) int { return a.factor.Cmp(b.factor) })
	return units, nil
}

// checkUnits checks the rules between the product's base unit and the
// alternate units of its variants: no alternate unit is the base unit, and
// no barcode repeats among the variants and their units. The database checks
// the barcodes against the rest of the tenant.
func checkUnits(base UnitRef, variants []*ProductVariant) error {
	seen := map[string]bool{}
	for _, v := range variants {
		if v.barcode != "" {
			seen[v.barcode] = true
		}
	}
	for _, v := range variants {
		for _, u := range v.units {
			if u.unit.ID == base.ID {
				return ErrAlternateUnitIsBase
			}
			if u.barcode == "" {
				continue
			}
			if seen[u.barcode] {
				return ErrBarcodeTaken
			}
			seen[u.barcode] = true
		}
	}
	return nil
}

// SetVariantUnit adds an alternate unit to a variant, or replaces it if the
// variant already has that unit. On error the product is unchanged.
func (p *Product) SetVariantUnit(variantID uuid.UUID, params VariantUnitParams) error {
	i, err := p.variantIndex(variantID)
	if err != nil {
		return err
	}
	current := p.variants[i]
	list := make([]VariantUnitParams, 0, len(current.units)+1)
	for _, u := range current.units {
		if u.unit.ID != params.Unit.ID {
			list = append(list, u.params())
		}
	}
	return p.replaceUnits(i, append(list, params))
}

// RemoveVariantUnit removes an alternate unit of a variant;
// ErrVariantUnitNotFound if the variant does not have it. Past sales keep
// the unit they were sold in.
func (p *Product) RemoveVariantUnit(variantID, unitID uuid.UUID) error {
	i, err := p.variantIndex(variantID)
	if err != nil {
		return err
	}
	if _, err := p.variants[i].Unit(unitID); err != nil {
		return err
	}
	var list []VariantUnitParams
	for _, u := range p.variants[i].units {
		if u.unit.ID != unitID {
			list = append(list, u.params())
		}
	}
	return p.replaceUnits(i, list)
}

// replaceUnits gives the variant at i the units in params if the product
// still follows the rules.
func (p *Product) replaceUnits(i int, params []VariantUnitParams) error {
	units, err := newVariantUnits(params)
	if err != nil {
		return err
	}
	v := *p.variants[i]
	v.units = units
	variants := slices.Clone(p.variants)
	variants[i] = &v
	if err := checkUnits(p.baseUnit, variants); err != nil {
		return err
	}
	p.variants = variants
	return nil
}

// PriceWithTax returns the unit's price plus IVA at the product's rate,
// rounded half-up to two decimals, as ProductVariant.PriceWithTax.
func (u VariantUnit) PriceWithTax(rate TaxRate) decimal.Decimal {
	return u.price.Mul(decimal.NewFromInt(1).Add(rate.Rate())).Round(2)
}
