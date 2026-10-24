package domain

import (
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
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
			if err := checkVariantPair(v, other); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkVariantPair checks two variants of a product against each other.
func checkVariantPair(v, other *ProductVariant) error {
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
