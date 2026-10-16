package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// VariantInput is the data of a variant added to a product.
type VariantInput struct {
	SKU        string
	Barcode    string
	Cost       decimal.Decimal
	Price      decimal.Decimal
	AxisValues map[string]string
}

// VariantUpdate is the editable data of an existing variant.
type VariantUpdate struct {
	VariantInput
	Active bool
}

// AddVariant adds an active variant to a product and returns the product.
// Its axis values must be among the tenant's axes.
func (s *Service) AddVariant(ctx context.Context, a Actor, productID uuid.UUID, in VariantInput) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "AddVariant", a, productAttr(productID))
	defer func() { endSpan(span, err) }()

	p, err := s.Products.GetByID(ctx, a.TenantID, productID)
	if err != nil {
		return nil, err
	}
	axes, err := s.resolveAxisValues(ctx, a, in.AxisValues)
	if err != nil {
		return nil, err
	}
	v, err := p.AddVariant(domain.VariantParams{
		SKU: in.SKU, Barcode: in.Barcode, Cost: in.Cost, Price: in.Price, AxisValues: axes,
	}, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(variantAttr(v.ID()))
	if err := s.Products.Update(ctx, p); err != nil {
		return nil, err
	}
	s.variantsCreated.Add(ctx, 1)
	return p, nil
}

// UpdateVariant replaces a variant's data and state and returns the
// product, with the axis values checked as AddVariant does. A new price
// follows ChangeVariantPrice: a variant of an inactive product keeps its
// price.
func (s *Service) UpdateVariant(ctx context.Context, a Actor, productID, variantID uuid.UUID, in VariantUpdate) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "UpdateVariant", a, productAttr(productID), variantAttr(variantID))
	defer func() { endSpan(span, err) }()

	p, err := s.Products.GetByID(ctx, a.TenantID, productID)
	if err != nil {
		return nil, err
	}
	v, err := p.Variant(variantID)
	if err != nil {
		return nil, err
	}
	axes, err := s.resolveAxisValues(ctx, a, in.AxisValues)
	if err != nil {
		return nil, err
	}
	priceChanged := !in.Price.Equal(v.Price())
	if err := p.UpdateVariant(variantID, domain.VariantDetails{
		SKU: in.SKU, Barcode: in.Barcode, Cost: in.Cost, AxisValues: axes,
	}); err != nil {
		return nil, err
	}
	if priceChanged {
		if err := p.ChangeVariantPrice(variantID, in.Price); err != nil {
			return nil, err
		}
	}
	if err := p.SetVariantActive(variantID, in.Active); err != nil {
		return nil, err
	}
	if err := s.Products.Update(ctx, p); err != nil {
		return nil, err
	}
	if priceChanged {
		s.priceChanges.Add(ctx, 1)
	}
	return p, nil
}
