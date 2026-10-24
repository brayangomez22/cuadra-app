package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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
