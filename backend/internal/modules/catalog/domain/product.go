package domain

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	maxSKULength         = 64
	maxBarcodeLength     = 64
	maxProductNameLength = 200
	maxDescriptionLength = 2000
)

var hundred = decimal.NewFromInt(100)

// Product is an item a tenant sells. Price is the taxable base, without IVA;
// PriceWithTax adds it. Its fields are private so it cannot be put in an
// invalid state.
type Product struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	sku         string
	barcode     string
	name        string
	description string
	categoryID  *uuid.UUID
	baseUnit    UnitOfMeasure
	cost        decimal.Decimal
	price       decimal.Decimal
	taxRate     TaxRate
	active      bool
	createdAt   time.Time
}

// NewProductParams holds the data of a new product. Barcode, Description and
// CategoryID are optional.
type NewProductParams struct {
	TenantID    uuid.UUID
	SKU         string
	Barcode     string
	Name        string
	Description string
	CategoryID  *uuid.UUID
	BaseUnit    UnitOfMeasure
	Cost        decimal.Decimal
	Price       decimal.Decimal
	TaxRate     TaxRate
}

// NewProduct creates an active product. The SKU is trimmed and uppercased so
// its uniqueness per tenant does not depend on letter case.
func NewProduct(p NewProductParams, now time.Time) (*Product, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newProduct(ProductSnapshot{
		ID:          id,
		TenantID:    p.TenantID,
		SKU:         p.SKU,
		Barcode:     p.Barcode,
		Name:        p.Name,
		Description: p.Description,
		CategoryID:  p.CategoryID,
		BaseUnit:    p.BaseUnit,
		Cost:        p.Cost,
		Price:       p.Price,
		TaxRate:     p.TaxRate,
		Active:      true,
		CreatedAt:   now,
	})
}

// newProduct checks the invariants shared by NewProduct and RehydrateProduct.
func newProduct(s ProductSnapshot) (*Product, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidProductID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	sku, err := normalizeSKU(s.SKU)
	if err != nil {
		return nil, err
	}
	barcode, err := normalizeBarcode(s.Barcode)
	if err != nil {
		return nil, err
	}
	name, err := normalizeText(s.Name, maxProductNameLength, ErrInvalidProductName)
	if err != nil {
		return nil, err
	}
	description := strings.TrimSpace(s.Description)
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return nil, ErrInvalidDescription
	}
	if s.CategoryID != nil && *s.CategoryID == uuid.Nil {
		return nil, ErrInvalidCategoryID
	}
	if !s.BaseUnit.valid() {
		return nil, ErrInvalidUnit
	}
	if !validPrice(s.Price) {
		return nil, ErrInvalidPrice
	}
	if s.Cost.IsNegative() || !fitsStorage(s.Cost) {
		return nil, ErrInvalidCost
	}
	return &Product{
		id:          s.ID,
		tenantID:    s.TenantID,
		sku:         sku,
		barcode:     barcode,
		name:        name,
		description: description,
		categoryID:  copyID(s.CategoryID),
		baseUnit:    s.BaseUnit,
		cost:        s.Cost,
		price:       s.Price,
		taxRate:     s.TaxRate,
		active:      s.Active,
		createdAt:   s.CreatedAt.UTC(),
	}, nil
}

// normalizeSKU trims and uppercases s; it must have no inner spaces nor
// control characters.
func normalizeSKU(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" || utf8.RuneCountInString(s) > maxSKULength {
		return "", ErrInvalidSKU
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", ErrInvalidSKU
		}
	}
	return s, nil
}

// normalizeBarcode trims s; empty means no barcode. Otherwise it must be
// printable ASCII without spaces. The EAN check digit is not verified because
// stores also print their own internal codes.
func normalizeBarcode(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) > maxBarcodeLength {
		return "", ErrInvalidBarcode
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '!' || s[i] > '~' {
			return "", ErrInvalidBarcode
		}
	}
	return s, nil
}

// validPrice reports whether d is a positive amount that fits storage.
func validPrice(d decimal.Decimal) bool { return d.IsPositive() && fitsStorage(d) }

// ID returns the product ID (UUID v7).
func (p *Product) ID() uuid.UUID { return p.id }

// TenantID returns the tenant the product belongs to.
func (p *Product) TenantID() uuid.UUID { return p.tenantID }

// SKU returns the normalized (uppercase) SKU.
func (p *Product) SKU() string { return p.sku }

// Barcode returns the barcode, or "" if the product has none.
func (p *Product) Barcode() string { return p.barcode }

// Name returns the product name.
func (p *Product) Name() string { return p.name }

// Description returns the description, possibly empty.
func (p *Product) Description() string { return p.description }

// CategoryID returns a copy of the category's ID, or nil if uncategorized.
func (p *Product) CategoryID() *uuid.UUID { return copyID(p.categoryID) }

// BaseUnit returns the unit the product is stocked and sold in.
func (p *Product) BaseUnit() UnitOfMeasure { return p.baseUnit }

// Cost returns the reference unit cost, without IVA.
func (p *Product) Cost() decimal.Decimal { return p.cost }

// Price returns the unit sale price, without IVA (taxable base).
func (p *Product) Price() decimal.Decimal { return p.price }

// TaxRate returns the IVA rate.
func (p *Product) TaxRate() TaxRate { return p.taxRate }

// IsActive reports whether the product can be sold and repriced.
func (p *Product) IsActive() bool { return p.active }

// CreatedAt returns the creation time in UTC.
func (p *Product) CreatedAt() time.Time { return p.createdAt }

// MarginPercent returns the margin over the sale price,
// (price − cost) / price × 100, rounded half-up to four decimals. It is
// negative when the product sells below cost.
func (p *Product) MarginPercent() decimal.Decimal {
	return p.price.Sub(p.cost).Mul(hundred).DivRound(p.price, amountScale)
}

// PriceWithTax returns the unit price plus IVA, rounded half-up to two
// decimals (COP). It is a display reference: sale totals compute IVA per
// line and per rate.
func (p *Product) PriceWithTax() decimal.Decimal {
	return p.price.Mul(decimal.NewFromInt(1).Add(p.taxRate.Rate())).Round(2)
}

// ChangePrice sets a new price. An inactive product cannot be repriced
// (ErrProductInactive): reactivate it first. On error the price is unchanged.
func (p *Product) ChangePrice(price decimal.Decimal) error {
	if !p.active {
		return ErrProductInactive
	}
	if !validPrice(price) {
		return ErrInvalidPrice
	}
	p.price = price
	return nil
}

// Deactivate hides the product from sales without deleting it, so past sales
// and stock movements keep their reference. Idempotent.
func (p *Product) Deactivate() { p.active = false }

// Activate makes an inactive product sellable again. Idempotent.
func (p *Product) Activate() { p.active = true }

// ProductSnapshot holds a stored product's fields, for RehydrateProduct.
type ProductSnapshot struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	SKU         string
	Barcode     string
	Name        string
	Description string
	CategoryID  *uuid.UUID
	BaseUnit    UnitOfMeasure
	Cost        decimal.Decimal
	Price       decimal.Decimal
	TaxRate     TaxRate
	Active      bool
	CreatedAt   time.Time
}

// RehydrateProduct rebuilds a stored product, checking the same invariants
// as NewProduct so corrupt data surfaces as an error instead of an invalid entity.
func RehydrateProduct(s ProductSnapshot) (*Product, error) { return newProduct(s) }

// ProductDetails are a product's editable data besides its price, which
// changes through ChangePrice.
type ProductDetails struct {
	SKU         string
	Barcode     string
	Name        string
	Description string
	CategoryID  *uuid.UUID
	BaseUnit    UnitOfMeasure
	Cost        decimal.Decimal
	TaxRate     TaxRate
}

// UpdateDetails replaces the product's details with the same rules as
// NewProduct. On error the product is left unchanged.
func (p *Product) UpdateDetails(d ProductDetails) error {
	updated, err := newProduct(ProductSnapshot{
		ID:          p.id,
		TenantID:    p.tenantID,
		SKU:         d.SKU,
		Barcode:     d.Barcode,
		Name:        d.Name,
		Description: d.Description,
		CategoryID:  d.CategoryID,
		BaseUnit:    d.BaseUnit,
		Cost:        d.Cost,
		Price:       p.price,
		TaxRate:     d.TaxRate,
		Active:      p.active,
		CreatedAt:   p.createdAt,
	})
	if err != nil {
		return err
	}
	*p = *updated
	return nil
}
