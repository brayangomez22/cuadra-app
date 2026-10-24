package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Page size bounds of SearchProducts.
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// ProductInput is the editable data of a product and of its default
// variant (SKU, barcode, cost and price).
type ProductInput struct {
	SKU         string
	Barcode     string
	Name        string
	Description string
	CategoryID  *uuid.UUID
	BaseUnitID  uuid.UUID
	Cost        decimal.Decimal
	Price       decimal.Decimal
	TaxRate     domain.TaxRate
	// Attributes are the product's values by attribute key. On an update
	// they replace the current ones: a key left out has no value.
	Attributes map[string]domain.AttributeValue
}

func (in ProductInput) details(unit domain.UnitRef, attrs domain.Attributes) domain.ProductDetails {
	return domain.ProductDetails{
		Name:        in.Name,
		Description: in.Description,
		CategoryID:  in.CategoryID,
		BaseUnit:    unit,
		TaxRate:     in.TaxRate,
		Attributes:  attrs,
	}
}

// CreateProduct stores a new, active product with its default variant.
func (s *Service) CreateProduct(ctx context.Context, a Actor, in ProductInput) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "CreateProduct", a)
	defer func() { usecase.End(span, err) }()

	unit, err := s.resolveUnit(ctx, a, in)
	if err != nil {
		return nil, err
	}
	attrs, err := s.resolveAttributes(ctx, a, in.Attributes)
	if err != nil {
		return nil, err
	}
	p, err := domain.NewProduct(domain.NewProductParams{
		TenantID:    a.TenantID,
		Name:        in.Name,
		Description: in.Description,
		CategoryID:  in.CategoryID,
		BaseUnit:    unit,
		TaxRate:     in.TaxRate,
		Attributes:  attrs,
		Variant: domain.VariantParams{
			SKU:     in.SKU,
			Barcode: in.Barcode,
			Cost:    in.Cost,
			Price:   in.Price,
		},
	}, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(productAttr(p.ID()), variantAttr(p.DefaultVariant().ID()))
	if err := s.requireCategory(ctx, a, p.CategoryID()); err != nil {
		return nil, err
	}
	if err := s.Products.Create(ctx, p); err != nil {
		return nil, err
	}
	s.productsCreated.Add(ctx, 1)
	return p, nil
}

// UpdateProduct replaces the product's data and its default variant's. A
// new price follows ChangeVariantPrice: an inactive product keeps its price.
// The default variant keeps its axis values, which change through
// UpdateVariant.
func (s *Service) UpdateProduct(ctx context.Context, a Actor, id uuid.UUID, in ProductInput) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "UpdateProduct", a, productAttr(id))
	defer func() { usecase.End(span, err) }()

	p, err := s.Products.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	unit, err := s.resolveUnit(ctx, a, in)
	if err != nil {
		return nil, err
	}
	attrs, err := s.resolveAttributes(ctx, a, in.Attributes)
	if err != nil {
		return nil, err
	}
	if err := p.UpdateDetails(in.details(unit, attrs)); err != nil {
		return nil, err
	}
	span.SetAttributes(variantAttr(p.DefaultVariant().ID()))
	priceChanged, err := updateDefaultVariant(p, in)
	if err != nil {
		return nil, err
	}
	if err := s.requireCategory(ctx, a, p.CategoryID()); err != nil {
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

// updateDefaultVariant replaces the default variant's data with in's, and
// its price when in has another one (priceChanged).
func updateDefaultVariant(p *domain.Product, in ProductInput) (priceChanged bool, err error) {
	def := p.DefaultVariant()
	if err := p.UpdateVariant(def.ID(), domain.VariantDetails{
		SKU: in.SKU, Barcode: in.Barcode, Cost: in.Cost, AxisValues: def.AxisValues(),
	}); err != nil {
		return false, err
	}
	if in.Price.Equal(def.Price()) {
		return false, nil
	}
	return true, p.ChangeVariantPrice(def.ID(), in.Price)
}

// resolveUnit returns the tenant's unit that in names;
// domain.ErrUnitNotFound if it does not exist in the tenant.
func (s *Service) resolveUnit(ctx context.Context, a Actor, in ProductInput) (domain.UnitRef, error) {
	u, err := s.Units.GetByID(ctx, a.TenantID, in.BaseUnitID)
	if err != nil {
		return domain.UnitRef{}, err
	}
	return u.Ref(), nil
}

// requireCategory checks that the category exists in the actor's tenant.
// The database checks it too; this gives the error before writing.
func (s *Service) requireCategory(ctx context.Context, a Actor, id *uuid.UUID) error {
	if id == nil {
		return nil
	}
	_, err := s.Categories.GetByID(ctx, a.TenantID, *id)
	return err
}

// GetProduct returns a product, active or not.
func (s *Service) GetProduct(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "GetProduct", a, productAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.Products.GetByID(ctx, a.TenantID, id)
}

// SearchProducts returns one page of the tenant's products. The page size
// defaults to 20 and is capped at 100. The searched text stays out of the
// span: it may be anything the user typed.
func (s *Service) SearchProducts(ctx context.Context, a Actor, f domain.ProductFilter) (_ domain.ProductPage, err error) {
	ctx, span := s.startSpan(ctx, "SearchProducts", a)
	defer func() { usecase.End(span, err) }()

	switch {
	case f.Limit <= 0:
		f.Limit = defaultPageSize
	case f.Limit > maxPageSize:
		f.Limit = maxPageSize
	}
	f.Offset = max(f.Offset, 0)

	page, err := s.Products.Search(ctx, a.TenantID, f)
	if err != nil {
		return domain.ProductPage{}, err
	}
	if f.Query != "" {
		result := "hit"
		if page.Total == 0 {
			result = "empty"
		}
		s.searches.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
	}
	return page, nil
}

// DeactivateProduct hides the product from sales. Idempotent.
func (s *Service) DeactivateProduct(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "DeactivateProduct", a, productAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.setActive(ctx, a, id, false)
}

// ActivateProduct makes the product sellable again. Idempotent.
func (s *Service) ActivateProduct(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Product, err error) {
	ctx, span := s.startSpan(ctx, "ActivateProduct", a, productAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.setActive(ctx, a, id, true)
}

func (s *Service) setActive(ctx context.Context, a Actor, id uuid.UUID, active bool) (*domain.Product, error) {
	p, err := s.Products.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if active {
		p.Activate()
	} else {
		p.Deactivate()
	}
	if err := s.Products.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
