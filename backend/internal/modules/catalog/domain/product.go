package domain

import (
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	maxProductNameLength = 200
	maxDescriptionLength = 2000
)

// Product groups the variants a tenant sells: it holds what they share
// (name, category, base unit, IVA rate) and the rules between them. It always
// has exactly one default variant, the one created with it; a simple product
// has only that one. Its attribute values are the tenant's own (brand,
// material...), checked against the tenant's definitions with
// ResolveAttributes. Its fields are private so it cannot be put in an invalid
// state.
type Product struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	name        string
	description string
	categoryID  *uuid.UUID
	baseUnit    UnitRef
	taxRate     TaxRate
	attributes  Attributes
	active      bool
	createdAt   time.Time
	variants    []*ProductVariant
}

// NewProductParams holds the data of a new product and of its default
// variant. Description, CategoryID and Attributes are optional.
type NewProductParams struct {
	TenantID    uuid.UUID
	Name        string
	Description string
	CategoryID  *uuid.UUID
	BaseUnit    UnitRef
	TaxRate     TaxRate
	Attributes  Attributes
	Variant     VariantParams
}

// NewProduct creates an active product with its default variant. Each one
// gets its own UUID v7.
func NewProduct(p NewProductParams, now time.Time) (*Product, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	variantID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newProduct(ProductSnapshot{
		ID:          id,
		TenantID:    p.TenantID,
		Name:        p.Name,
		Description: p.Description,
		CategoryID:  p.CategoryID,
		BaseUnit:    p.BaseUnit,
		TaxRate:     p.TaxRate,
		Attributes:  p.Attributes,
		Active:      true,
		CreatedAt:   now,
		Variants: []VariantSnapshot{{
			ID:         variantID,
			SKU:        p.Variant.SKU,
			Barcode:    p.Variant.Barcode,
			Cost:       p.Variant.Cost,
			Price:      p.Variant.Price,
			AxisValues: p.Variant.AxisValues,
			IsDefault:  true,
			Active:     true,
			CreatedAt:  now,
		}},
	})
}

// newProduct checks the invariants shared by NewProduct and RehydrateProduct.
func newProduct(s ProductSnapshot) (*Product, error) {
	p, err := productHeader(s)
	if err != nil {
		return nil, err
	}
	if len(s.Variants) == 0 {
		return nil, ErrInvalidVariants
	}
	variants := make([]*ProductVariant, 0, len(s.Variants))
	for _, vs := range s.Variants {
		v, err := newVariant(p.id, vs)
		if err != nil {
			return nil, err
		}
		variants = append(variants, v)
	}
	if err := checkVariants(variants); err != nil {
		return nil, err
	}
	if err := checkUnits(p.baseUnit, variants); err != nil {
		return nil, err
	}
	p.variants = variants
	return p, nil
}

// productHeader checks and builds the product's own fields, without variants.
func productHeader(s ProductSnapshot) (*Product, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidProductID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
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
	if s.BaseUnit.ID == uuid.Nil {
		return nil, ErrInvalidUnit
	}
	return &Product{
		id:          s.ID,
		tenantID:    s.TenantID,
		name:        name,
		description: description,
		categoryID:  copyID(s.CategoryID),
		baseUnit:    s.BaseUnit,
		taxRate:     s.TaxRate,
		attributes:  s.Attributes,
		active:      s.Active,
		createdAt:   s.CreatedAt.UTC(),
	}, nil
}

// checkVariants checks the rules between a product's variants: exactly one
// default, every variant with the same axes, no two with the same values,
// and no SKU or barcode repeated inside the product. The database checks the
// SKU and the barcode against the rest of the tenant.
func checkVariants(variants []*ProductVariant) error {
	defaults := 0
	for _, v := range variants {
		if v.isDefault {
			defaults++
		}
	}
	if defaults != 1 {
		return ErrInvalidVariants
	}
	for i, v := range variants {
		for _, other := range variants[:i] {
			switch {
			case !sameAxes(v.axisValues, other.axisValues):
				return ErrVariantAxesMismatch
			case maps.Equal(v.axisValues, other.axisValues):
				return ErrDuplicateVariant
			case v.sku == other.sku:
				return ErrSKUTaken
			case v.barcode != "" && v.barcode == other.barcode:
				return ErrBarcodeTaken
			}
		}
	}
	return nil
}

// ID returns the product ID (UUID v7).
func (p *Product) ID() uuid.UUID { return p.id }

// TenantID returns the tenant the product belongs to.
func (p *Product) TenantID() uuid.UUID { return p.tenantID }

// Name returns the product name.
func (p *Product) Name() string { return p.name }

// Description returns the description, possibly empty.
func (p *Product) Description() string { return p.description }

// CategoryID returns a copy of the category's ID, or nil if uncategorized.
func (p *Product) CategoryID() *uuid.UUID { return copyID(p.categoryID) }

// BaseUnit returns the unit the product is stocked and sold in.
func (p *Product) BaseUnit() UnitRef { return p.baseUnit }

// TaxRate returns the IVA rate of every variant.
func (p *Product) TaxRate() TaxRate { return p.taxRate }

// IsActive reports whether the product can be sold and repriced.
func (p *Product) IsActive() bool { return p.active }

// CreatedAt returns the creation time in UTC.
func (p *Product) CreatedAt() time.Time { return p.createdAt }

// Attributes returns the product's attribute values.
func (p *Product) Attributes() Attributes { return p.attributes }

// Variants returns the product's variants, in the order they were added.
func (p *Product) Variants() []*ProductVariant { return slices.Clone(p.variants) }

// DefaultVariant returns the variant created with the product.
func (p *Product) DefaultVariant() *ProductVariant {
	for _, v := range p.variants {
		if v.isDefault {
			return v
		}
	}
	// Unreachable: every constructor checks there is exactly one.
	panic("catalog: product without a default variant")
}

// Variant returns the variant with id, or ErrVariantNotFound if it is not
// one of the product's.
func (p *Product) Variant(id uuid.UUID) (*ProductVariant, error) {
	i, err := p.variantIndex(id)
	if err != nil {
		return nil, err
	}
	return p.variants[i], nil
}

func (p *Product) variantIndex(id uuid.UUID) (int, error) {
	i := slices.IndexFunc(p.variants, func(v *ProductVariant) bool { return v.id == id })
	if i < 0 {
		return 0, ErrVariantNotFound
	}
	return i, nil
}

// replaceVariant stores v at i if the variants still follow the rules. It
// copies the slice, so a failed change leaves the product as it was.
func (p *Product) replaceVariant(i int, v *ProductVariant) error {
	variants := slices.Clone(p.variants)
	variants[i] = v
	if err := checkVariants(variants); err != nil {
		return err
	}
	if err := checkUnits(p.baseUnit, variants); err != nil {
		return err
	}
	p.variants = variants
	return nil
}

// AddVariant adds an active variant with its own UUID v7. It must name the
// same axes as the others with different values; a simple product gets axes
// on its default variant first (UpdateVariant). On error the product is
// unchanged.
func (p *Product) AddVariant(params VariantParams, now time.Time) (*ProductVariant, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	v, err := newVariant(p.id, VariantSnapshot{
		ID: id, SKU: params.SKU, Barcode: params.Barcode, Cost: params.Cost, Price: params.Price,
		AxisValues: params.AxisValues, Active: true, CreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	variants := append(slices.Clone(p.variants), v)
	if err := checkVariants(variants); err != nil {
		return nil, err
	}
	if err := checkUnits(p.baseUnit, variants); err != nil {
		return nil, err
	}
	p.variants = variants
	return v, nil
}

// UpdateVariant replaces a variant's details with the same rules as
// AddVariant. On error the product is unchanged.
func (p *Product) UpdateVariant(id uuid.UUID, d VariantDetails) error {
	i, err := p.variantIndex(id)
	if err != nil {
		return err
	}
	s := p.variants[i].snapshot()
	s.SKU, s.Barcode, s.Cost, s.AxisValues = d.SKU, d.Barcode, d.Cost, d.AxisValues
	v, err := newVariant(p.id, s)
	if err != nil {
		return err
	}
	return p.replaceVariant(i, v)
}

// ChangeVariantPrice sets a variant's price. A variant of an inactive
// product cannot be repriced (ErrProductInactive): reactivate it first. On
// error the price is unchanged.
func (p *Product) ChangeVariantPrice(id uuid.UUID, price decimal.Decimal) error {
	i, err := p.variantIndex(id)
	if err != nil {
		return err
	}
	if !p.active {
		return ErrProductInactive
	}
	if !validPrice(price) {
		return ErrInvalidPrice
	}
	v := *p.variants[i]
	v.price = price
	return p.replaceVariant(i, &v)
}

// SetVariantActive makes a variant sellable or hides it from sales, without
// changing the product. Idempotent.
func (p *Product) SetVariantActive(id uuid.UUID, active bool) error {
	i, err := p.variantIndex(id)
	if err != nil {
		return err
	}
	v := *p.variants[i]
	v.active = active
	return p.replaceVariant(i, &v)
}

// Deactivate hides the product and all its variants from sales without
// deleting it, so past sales and stock movements keep their reference.
// Idempotent.
func (p *Product) Deactivate() { p.active = false }

// Activate makes an inactive product sellable again. Idempotent.
func (p *Product) Activate() { p.active = true }

// ProductSnapshot holds a stored product's fields and its variants, for
// RehydrateProduct.
type ProductSnapshot struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	CategoryID  *uuid.UUID
	BaseUnit    UnitRef
	TaxRate     TaxRate
	Attributes  Attributes
	Active      bool
	CreatedAt   time.Time
	Variants    []VariantSnapshot
}

// RehydrateProduct rebuilds a stored product, checking the same invariants
// as NewProduct so corrupt data surfaces as an error instead of an invalid entity.
func RehydrateProduct(s ProductSnapshot) (*Product, error) { return newProduct(s) }

// ProductDetails are the product's own editable data. Each variant's data
// changes through UpdateVariant and ChangeVariantPrice.
type ProductDetails struct {
	Name        string
	Description string
	CategoryID  *uuid.UUID
	BaseUnit    UnitRef
	TaxRate     TaxRate
	Attributes  Attributes
}

// UpdateDetails replaces the product's details with the same rules as
// NewProduct. On error the product is left unchanged.
func (p *Product) UpdateDetails(d ProductDetails) error {
	updated, err := productHeader(ProductSnapshot{
		ID:          p.id,
		TenantID:    p.tenantID,
		Name:        d.Name,
		Description: d.Description,
		CategoryID:  d.CategoryID,
		BaseUnit:    d.BaseUnit,
		TaxRate:     d.TaxRate,
		Attributes:  d.Attributes,
		Active:      p.active,
		CreatedAt:   p.createdAt,
	})
	if err != nil {
		return err
	}
	if err := checkUnits(updated.baseUnit, p.variants); err != nil {
		return err
	}
	updated.variants = p.variants
	*p = *updated
	return nil
}
