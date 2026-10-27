package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ProductWithVariantsInput is a new product with all its variants; the
// first one is its default variant.
type ProductWithVariantsInput struct {
	Name        string
	Description string
	CategoryID  *uuid.UUID
	BaseUnitID  uuid.UUID
	TaxRate     domain.TaxRate
	Attributes  map[string]domain.AttributeValue
	Variants    []VariantInput
}

// CreateProductWithVariants stores a new, active product with its variants
// in one operation: either all of it is stored or nothing is. The axis
// values must be among the tenant's axes.
func (s *Service) CreateProductWithVariants(ctx context.Context, a Actor, in ProductWithVariantsInput) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "CreateProductWithVariants", a)
	defer func() { usecase.End(span, err) }()

	u, err := s.Units.GetByID(ctx, a.TenantID, in.BaseUnitID)
	if err != nil {
		return nil, err
	}
	attrs, err := s.resolveAttributes(ctx, a, in.Attributes)
	if err != nil {
		return nil, err
	}
	axes, err := s.VariantAxes.List(ctx, a.TenantID)
	if err != nil {
		return nil, err
	}
	p, err := newWithVariants(domain.NewProductParams{
		TenantID: a.TenantID, Name: in.Name, Description: in.Description, CategoryID: in.CategoryID,
		BaseUnit: u.Ref(), TaxRate: in.TaxRate, Attributes: attrs,
	}, in.Variants, axes, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(productAttr(p.ID()))
	if err := s.requireCategory(ctx, a, p.CategoryID()); err != nil {
		return nil, err
	}
	if err := s.Products.Create(ctx, p); err != nil {
		return nil, err
	}
	s.productsCreated.Add(ctx, 1)
	return p, nil
}

// newWithVariants builds a new product whose default variant is
// variants[0] and adds the others, with their axis values resolved against
// the tenant's axes.
func newWithVariants(params domain.NewProductParams, variants []VariantInput, axes []*domain.VariantAxis, now time.Time) (*domain.Product, error) {
	if len(variants) == 0 {
		return nil, domain.ErrNoVariants
	}
	first, err := variantParams(variants[0], axes)
	if err != nil {
		return nil, err
	}
	params.Variant = first
	p, err := domain.NewProduct(params, now)
	if err != nil {
		return nil, err
	}
	for _, in := range variants[1:] {
		vp, err := variantParams(in, axes)
		if err != nil {
			return nil, err
		}
		if _, err := p.AddVariant(vp, now); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// variantParams resolves in's axis values against the tenant's axes.
func variantParams(in VariantInput, axes []*domain.VariantAxis) (domain.VariantParams, error) {
	values, err := domain.ResolveAxisValues(axes, in.AxisValues)
	if err != nil {
		return domain.VariantParams{}, err
	}
	return domain.VariantParams{SKU: in.SKU, Barcode: in.Barcode, Cost: in.Cost, Price: in.Price, AxisValues: values}, nil
}

// requireUnitChangeable fails with domain.ErrBaseUnitLocked when unit is
// not p's base unit and a variant of p has stock movements: they are
// counted in the current base unit.
func (s *Service) requireUnitChangeable(ctx context.Context, a Actor, p *domain.Product, unit domain.UnitRef) error {
	if unit.ID == p.BaseUnit().ID {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(p.Variants()))
	for _, v := range p.Variants() {
		ids = append(ids, v.ID())
	}
	moved, err := s.History.HasMovements(ctx, a.TenantID, ids)
	if err != nil {
		return err
	}
	if moved {
		return domain.ErrBaseUnitLocked
	}
	return nil
}
