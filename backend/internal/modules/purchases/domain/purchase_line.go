package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	maxProductNameLength = 200
	maxUnitCodeLength    = 20
)

// LineUnit is the unit a line was bought in: an alternate unit of the
// variant (ID and its factor: how many base units it holds) or, with a nil
// ID, the base unit (factor 1). Code is how the invoice names it.
type LineUnit struct {
	ID     *uuid.UUID
	Code   string
	Factor decimal.Decimal
}

func (u LineUnit) check() error {
	if !u.Factor.IsPositive() || !fitsStorage(u.Factor) || len([]rune(u.Code)) > maxUnitCodeLength {
		return ErrInvalidUnit
	}
	if (u.ID == nil && !u.Factor.Equal(decimal.NewFromInt(1))) || (u.ID != nil && *u.ID == uuid.Nil) {
		return ErrInvalidUnit
	}
	return nil
}

func (u LineUnit) clone() LineUnit {
	if u.ID != nil {
		id := *u.ID
		u.ID = &id
	}
	return u
}

// LineInput is a line of the supplier's invoice: a quantity of a variant in
// a purchase unit, at a unit cost in that unit before VAT, and its VAT rate.
// The names are the catalog's when the purchase is registered.
type LineInput struct {
	VariantID    uuid.UUID
	ProductName  string
	VariantLabel string
	Unit         LineUnit
	Quantity     decimal.Decimal
	UnitCost     decimal.Decimal
	TaxRate      decimal.Decimal
}

// Received is what a line put into the stock, in the base unit: its
// quantity × factor at its cost ÷ factor, as the inventory converted it.
type Received struct {
	BaseQuantity decimal.Decimal
	BaseUnitCost decimal.Decimal
}

// Line is a line of a purchase. It keeps the unit it was bought in, with the
// quantity and cost in that unit, and what it put into the stock in the
// base unit (once received).
type Line struct {
	in       LineInput
	subtotal decimal.Decimal
	tax      decimal.Decimal
	received Received
}

// newLine checks in and computes its amounts: subtotal = quantity × cost
// and VAT = subtotal × rate, each rounded half-up to two decimals.
func newLine(in LineInput) (Line, error) {
	if in.VariantID == uuid.Nil {
		return Line{}, ErrInvalidVariantID
	}
	var err error
	if in.ProductName, err = requiredText(in.ProductName, maxProductNameLength, ErrInvalidProductName); err != nil {
		return Line{}, err
	}
	if in.VariantLabel, err = optionalText(in.VariantLabel, maxProductNameLength, ErrInvalidProductName); err != nil {
		return Line{}, err
	}
	if err := in.Unit.check(); err != nil {
		return Line{}, err
	}
	if err := checkLineAmounts(in); err != nil {
		return Line{}, err
	}
	in.Unit = in.Unit.clone()
	subtotal := roundMoney(in.Quantity.Mul(in.UnitCost))
	if !fitsStorage(subtotal) {
		return Line{}, ErrInvalidUnitCost
	}
	return Line{in: in, subtotal: subtotal, tax: roundMoney(subtotal.Mul(in.TaxRate))}, nil
}

func checkLineAmounts(in LineInput) error {
	if !in.Quantity.IsPositive() || !fitsStorage(in.Quantity) {
		return ErrInvalidQuantity
	}
	if in.UnitCost.IsNegative() || !fitsStorage(in.UnitCost) {
		return ErrInvalidUnitCost
	}
	if in.TaxRate.IsNegative() || in.TaxRate.GreaterThanOrEqual(decimal.NewFromInt(1)) || !fitsStorage(in.TaxRate) {
		return ErrInvalidTaxRate
	}
	return nil
}

func (r Received) valid() bool {
	return r.BaseQuantity.IsPositive() && fitsStorage(r.BaseQuantity) &&
		!r.BaseUnitCost.IsNegative() && fitsStorage(r.BaseUnitCost)
}

// VariantID returns the variant bought.
func (l Line) VariantID() uuid.UUID { return l.in.VariantID }

// ProductName returns the product's name when it was bought.
func (l Line) ProductName() string { return l.in.ProductName }

// VariantLabel returns the variant's label when it was bought ("" for a
// product's only variant).
func (l Line) VariantLabel() string { return l.in.VariantLabel }

// Unit returns the unit the line was bought in.
func (l Line) Unit() LineUnit { return l.in.Unit.clone() }

// Quantity returns the quantity in the purchase unit.
func (l Line) Quantity() decimal.Decimal { return l.in.Quantity }

// UnitCost returns the cost of one purchase unit, before VAT.
func (l Line) UnitCost() decimal.Decimal { return l.in.UnitCost }

// TaxRate returns the VAT rate (0.19).
func (l Line) TaxRate() decimal.Decimal { return l.in.TaxRate }

// Subtotal returns quantity × cost, before VAT.
func (l Line) Subtotal() decimal.Decimal { return l.subtotal }

// Tax returns the line's VAT.
func (l Line) Tax() decimal.Decimal { return l.tax }

// Total returns the subtotal plus VAT.
func (l Line) Total() decimal.Decimal { return l.subtotal.Add(l.tax) }

// BaseQuantity returns the quantity that entered the stock, in the base unit.
func (l Line) BaseQuantity() decimal.Decimal { return l.received.BaseQuantity }

// BaseUnitCost returns the cost of one base unit that fed the average cost.
func (l Line) BaseUnitCost() decimal.Decimal { return l.received.BaseUnitCost }
