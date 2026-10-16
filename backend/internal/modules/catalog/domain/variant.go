package domain

import (
	"maps"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	maxSKULength       = 64
	maxBarcodeLength   = 64
	maxAxes            = 5
	maxAxisNameLength  = 50
	maxAxisValueLength = 100
)

var hundred = decimal.NewFromInt(100)

// ProductVariant is what a tenant sells and counts: a product in one
// combination of its axes (size, color, presentation...). A simple product
// has a single default variant without axes. Price is the taxable base,
// without IVA. It changes only through its Product, which keeps the rules
// that involve the other variants.
type ProductVariant struct {
	id         uuid.UUID
	productID  uuid.UUID
	sku        string
	barcode    string
	cost       decimal.Decimal
	price      decimal.Decimal
	axisValues map[string]string
	isDefault  bool
	active     bool
	createdAt  time.Time
}

// VariantParams holds the data of a new variant. Barcode and AxisValues are
// optional.
type VariantParams struct {
	SKU     string
	Barcode string
	Cost    decimal.Decimal
	Price   decimal.Decimal
	// AxisValues maps each axis to this variant's value: {"color": "Azul"}.
	// The tenant names the axes.
	AxisValues map[string]string
}

// VariantDetails are a variant's editable data besides its price, which
// changes through Product.ChangeVariantPrice.
type VariantDetails struct {
	SKU        string
	Barcode    string
	Cost       decimal.Decimal
	AxisValues map[string]string
}

// VariantSnapshot holds a stored variant's fields, for RehydrateProduct.
type VariantSnapshot struct {
	ID         uuid.UUID
	SKU        string
	Barcode    string
	Cost       decimal.Decimal
	Price      decimal.Decimal
	AxisValues map[string]string
	IsDefault  bool
	Active     bool
	CreatedAt  time.Time
}

// newVariant checks a variant's own invariants; the ones involving the other
// variants are the product's.
func newVariant(productID uuid.UUID, s VariantSnapshot) (*ProductVariant, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidVariantID
	}
	sku, err := normalizeSKU(s.SKU)
	if err != nil {
		return nil, err
	}
	barcode, err := normalizeBarcode(s.Barcode)
	if err != nil {
		return nil, err
	}
	if !validPrice(s.Price) {
		return nil, ErrInvalidPrice
	}
	if s.Cost.IsNegative() || !fitsStorage(s.Cost) {
		return nil, ErrInvalidCost
	}
	axes, err := normalizeAxes(s.AxisValues)
	if err != nil {
		return nil, err
	}
	return &ProductVariant{
		id:         s.ID,
		productID:  productID,
		sku:        sku,
		barcode:    barcode,
		cost:       s.Cost,
		price:      s.Price,
		axisValues: axes,
		isDefault:  s.IsDefault,
		active:     s.Active,
		createdAt:  s.CreatedAt.UTC(),
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

// normalizeAxes trims names and values and lowercases names, so "Talla" and
// "talla" are one axis. Values keep their case: they are what the receipt
// shows. Neither may be empty, and two names may not collide once normalized.
func normalizeAxes(in map[string]string) (map[string]string, error) {
	if len(in) > maxAxes {
		return nil, ErrInvalidAxisValues
	}
	out := make(map[string]string, len(in))
	for name, value := range in {
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		if name == "" || utf8.RuneCountInString(name) > maxAxisNameLength ||
			value == "" || utf8.RuneCountInString(value) > maxAxisValueLength {
			return nil, ErrInvalidAxisValues
		}
		if _, dup := out[name]; dup {
			return nil, ErrInvalidAxisValues
		}
		out[name] = value
	}
	return out, nil
}

// validPrice reports whether d is a positive amount that fits storage.
func validPrice(d decimal.Decimal) bool { return d.IsPositive() && fitsStorage(d) }

// sameAxes reports whether a and b name the same axes, whatever their values.
func sameAxes(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for name := range a {
		if _, ok := b[name]; !ok {
			return false
		}
	}
	return true
}

// ID returns the variant ID. A product created before variants existed
// reuses its own ID for its default variant; never assume either way.
func (v *ProductVariant) ID() uuid.UUID { return v.id }

// ProductID returns the product the variant belongs to.
func (v *ProductVariant) ProductID() uuid.UUID { return v.productID }

// SKU returns the normalized (uppercase) SKU.
func (v *ProductVariant) SKU() string { return v.sku }

// Barcode returns the barcode, or "" if the variant has none.
func (v *ProductVariant) Barcode() string { return v.barcode }

// Cost returns the reference unit cost, without IVA.
func (v *ProductVariant) Cost() decimal.Decimal { return v.cost }

// Price returns the unit sale price, without IVA (taxable base).
func (v *ProductVariant) Price() decimal.Decimal { return v.price }

// AxisValues returns a copy of the variant's value per axis; empty for the
// default variant of a simple product.
func (v *ProductVariant) AxisValues() map[string]string { return maps.Clone(v.axisValues) }

// Label names the variant by its axis values, ordered by axis name and
// joined by " · " ({"talla": "M", "color": "Rojo"} is "Rojo · M"), so it
// is the same every time; "" for a variant without axes. A sale keeps the
// label of the variant it sold.
func (v *ProductVariant) Label() string {
	names := slices.Sorted(maps.Keys(v.axisValues))
	values := make([]string, 0, len(names))
	for _, name := range names {
		values = append(values, v.axisValues[name])
	}
	return strings.Join(values, " · ")
}

// IsDefault reports whether this is the variant created with the product.
func (v *ProductVariant) IsDefault() bool { return v.isDefault }

// IsActive reports whether the variant can be sold. The product must be
// active too.
func (v *ProductVariant) IsActive() bool { return v.active }

// CreatedAt returns the creation time in UTC.
func (v *ProductVariant) CreatedAt() time.Time { return v.createdAt }

// MarginPercent returns the margin over the sale price,
// (price − cost) / price × 100, rounded half-up to four decimals. It is
// negative when the variant sells below cost.
func (v *ProductVariant) MarginPercent() decimal.Decimal {
	return v.price.Sub(v.cost).Mul(hundred).DivRound(v.price, amountScale)
}

// PriceWithTax returns the unit price plus IVA at the product's rate,
// rounded half-up to two decimals (COP). It is a display reference: sale
// totals compute IVA per line and per rate.
func (v *ProductVariant) PriceWithTax(rate TaxRate) decimal.Decimal {
	return v.price.Mul(decimal.NewFromInt(1).Add(rate.Rate())).Round(2)
}

func (v *ProductVariant) snapshot() VariantSnapshot {
	return VariantSnapshot{
		ID: v.id, SKU: v.sku, Barcode: v.barcode, Cost: v.cost, Price: v.price,
		AxisValues: v.axisValues, IsDefault: v.isDefault, Active: v.active, CreatedAt: v.createdAt,
	}
}
