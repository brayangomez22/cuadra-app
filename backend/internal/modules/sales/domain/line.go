package domain

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const maxProductNameLength = 200

// LineInput is a line to add to a sale. ProductName, UnitPrice and TaxRate
// are the catalog's at the time of the sale; the line keeps them so the sale
// does not change if the product does.
type LineInput struct {
	ProductID   uuid.UUID
	ProductName string
	// Quantity is in the product's base unit; it can be fractional (2.5 m).
	Quantity decimal.Decimal
	// UnitPrice is without IVA, as the catalog stores it.
	UnitPrice decimal.Decimal
	// TaxRate is the IVA rate as a fraction (0.19).
	TaxRate decimal.Decimal
	// Discount is an amount of COP off the line, before IVA.
	Discount decimal.Decimal
}

// SaleLine is a product sold and its amounts:
//
//	Gross = round2(Quantity × UnitPrice)
//	Base  = Gross − Discount
//	Tax   = round2(Base × TaxRate)
//	Total = Base + Tax
//
// Rounding is half-up. Each line's amounts are final, so the totals by rate
// add up exactly to the lines, as the electronic invoice requires.
type SaleLine struct {
	productID   uuid.UUID
	productName string
	quantity    decimal.Decimal
	unitPrice   decimal.Decimal
	taxRate     decimal.Decimal
	discount    decimal.Decimal
	gross       decimal.Decimal
	tax         decimal.Decimal
}

// newLine validates in and computes the line's amounts. The discount must be
// within limit.
func newLine(in LineInput, limit DiscountLimit) (SaleLine, error) {
	if in.ProductID == uuid.Nil {
		return SaleLine{}, ErrInvalidProductID
	}
	name := strings.TrimSpace(in.ProductName)
	if name == "" || utf8.RuneCountInString(name) > maxProductNameLength {
		return SaleLine{}, ErrInvalidProductName
	}
	if !in.Quantity.IsPositive() || !fitsStorage(in.Quantity) {
		return SaleLine{}, ErrInvalidQuantity
	}
	if !in.UnitPrice.IsPositive() || !fitsStorage(in.UnitPrice) {
		return SaleLine{}, ErrInvalidUnitPrice
	}
	if in.TaxRate.IsNegative() || in.TaxRate.GreaterThanOrEqual(decimal.NewFromInt(1)) || !fitsStorage(in.TaxRate) {
		return SaleLine{}, ErrInvalidTaxRate
	}
	gross := roundMoney(in.Quantity.Mul(in.UnitPrice))
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
		productID:   in.ProductID,
		productName: name,
		quantity:    in.Quantity,
		unitPrice:   in.UnitPrice,
		taxRate:     in.TaxRate,
		discount:    in.Discount,
		gross:       gross,
		tax:         roundMoney(base.Mul(in.TaxRate)),
	}, nil
}

// ProductID returns the product sold.
func (l SaleLine) ProductID() uuid.UUID { return l.productID }

// ProductName returns the product's name at the time of the sale.
func (l SaleLine) ProductName() string { return l.productName }

// Quantity returns the quantity sold, in the product's base unit.
func (l SaleLine) Quantity() decimal.Decimal { return l.quantity }

// UnitPrice returns the price per unit without IVA.
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
