package domain

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	maxProductNameLength  = 200
	maxVariantLabelLength = 200
)

// LineInput is a line to add to a sale: a variant of a product. ProductName,
// VariantLabel and TaxRate are the catalog's at the time of the sale, and
// UnitPrice is what the price rule PriceRule gave; the line keeps them so
// the sale does not change if the product does.
type LineInput struct {
	VariantID uuid.UUID
	// ProductID is the variant's product, kept for the reports by product.
	ProductID   uuid.UUID
	ProductName string
	// VariantLabel names the variant ("Rojo · M"); empty for a variant
	// without axes, as a simple product's.
	VariantLabel string
	// Quantity is in Unit; it can be fractional (2.5 m).
	Quantity decimal.Decimal
	// Unit is the unit sold: the product's base unit (the zero value) or an
	// alternate one, whose quantity in the base unit must fit four decimals.
	Unit LineUnit
	// UnitPrice is the price of one Unit, without IVA. A price rule may give it with more than four
	// decimals: the line rounds it, half-up, to the four it stores.
	UnitPrice decimal.Decimal
	// PriceRule is the rule that gave UnitPrice.
	PriceRule PriceRuleID
	// TaxRate is the IVA rate as a fraction (0.19).
	TaxRate decimal.Decimal
	// Discount is an amount of COP off the line, before IVA.
	Discount decimal.Decimal
}

// SaleLine is a variant of a product sold and its amounts:
//
//	Gross = round2(Quantity × UnitPrice)
//	Base  = Gross − Discount
//	Tax   = round2(Base × TaxRate)
//	Total = Base + Tax
//
// Rounding is half-up. Each line's amounts are final, so the totals by rate
// add up exactly to the lines, as the electronic invoice requires.
type SaleLine struct {
	variantID    uuid.UUID
	productID    uuid.UUID
	productName  string
	variantLabel string
	priceRule    PriceRuleID
	unit         LineUnit
	quantity     decimal.Decimal
	unitPrice    decimal.Decimal
	taxRate      decimal.Decimal
	discount     decimal.Decimal
	gross        decimal.Decimal
	tax          decimal.Decimal
}

// newLine validates in and computes the line's amounts. The discount must be
// within limit.
func newLine(in LineInput, limit DiscountLimit) (SaleLine, error) {
	name, label, err := lineNames(in)
	if err != nil {
		return SaleLine{}, err
	}
	rule, err := NewPriceRuleID(string(in.PriceRule))
	if err != nil {
		return SaleLine{}, err
	}
	unit, unitPrice, err := linePricing(in)
	if err != nil {
		return SaleLine{}, err
	}
	gross := roundMoney(in.Quantity.Mul(unitPrice))
	if !fitsStorage(gross) {
		return SaleLine{}, ErrInvalidQuantity
	}
	if in.Discount.IsNegative() || !isMoney(in.Discount) || in.Discount.GreaterThan(gross) {
		return SaleLine{}, ErrInvalidDiscount
	}
	if !limit.allows(in.Discount, gross) {
		return SaleLine{}, ErrDiscountExceedsLimit
	}
	base := gross.Sub(in.Discount)
	return SaleLine{
		variantID:    in.VariantID,
		productID:    in.ProductID,
		productName:  name,
		variantLabel: label,
		priceRule:    rule,
		unit:         unit,
		quantity:     in.Quantity,
		unitPrice:    unitPrice,
		taxRate:      in.TaxRate,
		discount:     in.Discount,
		gross:        gross,
		tax:          roundMoney(base.Mul(in.TaxRate)),
	}, nil
}

// lineNames checks what the line sells (variant and product) and returns
// the product's name and the variant's label, trimmed.
func lineNames(in LineInput) (name, label string, err error) {
	if in.VariantID == uuid.Nil {
		return "", "", ErrInvalidVariantID
	}
	if in.ProductID == uuid.Nil {
		return "", "", ErrInvalidProductID
	}
	name = strings.TrimSpace(in.ProductName)
	if name == "" || utf8.RuneCountInString(name) > maxProductNameLength {
		return "", "", ErrInvalidProductName
	}
	label = strings.TrimSpace(in.VariantLabel)
	if utf8.RuneCountInString(label) > maxVariantLabelLength {
		return "", "", ErrInvalidVariantLabel
	}
	return name, label, nil
}

// linePricing checks the quantity, the unit, the unit price and the IVA rate
// of the line, and returns the unit and the unit price rounded.
func linePricing(in LineInput) (LineUnit, decimal.Decimal, error) {
	if !in.Quantity.IsPositive() || !fitsStorage(in.Quantity) {
		return LineUnit{}, decimal.Decimal{}, ErrInvalidQuantity
	}
	unit, err := newLineUnit(in.Unit)
	if err != nil {
		return LineUnit{}, decimal.Decimal{}, err
	}
	if !fitsStorage(in.Quantity.Mul(unit.Factor)) {
		return LineUnit{}, decimal.Decimal{}, ErrQuantityPrecision
	}
	unitPrice := in.UnitPrice.Round(amountScale)
	if !unitPrice.IsPositive() || !fitsStorage(unitPrice) {
		return LineUnit{}, decimal.Decimal{}, ErrInvalidUnitPrice
	}
	if in.TaxRate.IsNegative() || in.TaxRate.GreaterThanOrEqual(decimal.NewFromInt(1)) || !fitsStorage(in.TaxRate) {
		return LineUnit{}, decimal.Decimal{}, ErrInvalidTaxRate
	}
	return unit, unitPrice, nil
}

// VariantID returns the variant sold: what the stock counts.
func (l SaleLine) VariantID() uuid.UUID { return l.variantID }

// ProductID returns the product of the variant sold.
func (l SaleLine) ProductID() uuid.UUID { return l.productID }

// ProductName returns the product's name at the time of the sale.
func (l SaleLine) ProductName() string { return l.productName }

// VariantLabel returns the variant's label at the time of the sale ("Rojo ·
// M"), or "" for a variant without axes.
func (l SaleLine) VariantLabel() string { return l.variantLabel }

// PriceRule returns the rule that gave the line's unit price.
func (l SaleLine) PriceRule() PriceRuleID { return l.priceRule }

// Quantity returns the quantity sold, in the line's Unit.
func (l SaleLine) Quantity() decimal.Decimal { return l.quantity }

// UnitPrice returns the price of one Unit without IVA.
func (l SaleLine) UnitPrice() decimal.Decimal { return l.unitPrice }

// TaxRate returns the IVA rate as a fraction.
func (l SaleLine) TaxRate() decimal.Decimal { return l.taxRate }

// Discount returns the amount off the line, before IVA.
func (l SaleLine) Discount() decimal.Decimal { return l.discount }

// Gross returns quantity × unit price, rounded to two decimals.
func (l SaleLine) Gross() decimal.Decimal { return l.gross }

// Base returns the taxable amount: gross minus discount.
func (l SaleLine) Base() decimal.Decimal { return l.gross.Sub(l.discount) }

// Tax returns the line's IVA, rounded to two decimals.
func (l SaleLine) Tax() decimal.Decimal { return l.tax }

// Total returns what the customer pays for the line: base plus IVA.
func (l SaleLine) Total() decimal.Decimal { return l.Base().Add(l.tax) }
