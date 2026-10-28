package app

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// createDraft builds the new product d describes and, when save is true,
// stores it with its category.
func (s *Service) createDraft(ctx context.Context, a Actor, c *draftCatalog, d ProductDraft, save bool) DraftResult {
	details, err := s.draftDetails(ctx, a, c, d, save)
	if err != nil {
		return DraftResult{Err: err}
	}
	p, err := newWithVariants(domain.NewProductParams{
		TenantID: a.TenantID, Name: details.Name, Description: details.Description,
		CategoryID: details.CategoryID, BaseUnit: details.BaseUnit, TaxRate: details.TaxRate,
		Attributes: details.Attributes,
	}, d.Variants, c.axes, s.Now())
	if err != nil {
		return DraftResult{Err: err}
	}
	if save {
		if err := s.Products.Create(ctx, p); err != nil {
			return DraftResult{Err: err}
		}
		s.productsCreated.Add(ctx, 1)
		c.remember(p)
	}
	r := DraftResult{ProductID: p.ID(), Outcome: DraftCreated}
	for _, v := range p.Variants() {
		r.Variants = append(r.Variants, DraftVariant{ID: v.ID(), Created: true})
	}
	return r
}

// updateDraft replaces the details of existing with d's, updates the
// variants whose SKU it has and adds the others, and when save is true
// stores it.
func (s *Service) updateDraft(ctx context.Context, a Actor, c *draftCatalog, existing *domain.Product, d ProductDraft, save bool) DraftResult {
	details, err := s.draftDetails(ctx, a, c, d, save)
	if err != nil {
		return DraftResult{Err: err}
	}
	if err := s.requireUnitChangeable(ctx, a, existing, details.BaseUnit); err != nil {
		return DraftResult{Err: err}
	}
	if err := existing.UpdateDetails(details); err != nil {
		return DraftResult{Err: err}
	}
	r := DraftResult{ProductID: existing.ID(), Outcome: DraftUpdated}
	prices, added := 0, 0
	for _, in := range d.Variants {
		v, priceChanged, created, err := s.draftVariant(c, existing, in)
		if err != nil {
			return DraftResult{Err: err}
		}
		r.Variants = append(r.Variants, DraftVariant{ID: v.ID(), Created: created})
		prices, added = prices+boolCount(priceChanged), added+boolCount(created)
	}
	if !save {
		return r
	}
	if err := s.Products.Update(ctx, existing); err != nil {
		return DraftResult{Err: err}
	}
	s.priceChanges.Add(ctx, int64(prices))
	s.variantsCreated.Add(ctx, int64(added))
	return r
}

// draftVariant updates the variant of p with in's SKU, or adds it.
func (s *Service) draftVariant(c *draftCatalog, p *domain.Product, in VariantInput) (_ *domain.ProductVariant, priceChanged, created bool, err error) {
	params, err := variantParams(in, c.axes)
	if err != nil {
		return nil, false, false, err
	}
	v, ok := variantWithSKU(p, in.SKU)
	if !ok {
		v, err = p.AddVariant(params, s.Now())
		return v, false, true, err
	}
	if err := p.UpdateVariant(v.ID(), domain.VariantDetails{
		SKU: params.SKU, Barcode: params.Barcode, Cost: params.Cost, AxisValues: params.AxisValues,
	}); err != nil {
		return nil, false, false, err
	}
	if params.Price.Equal(v.Price()) {
		return v, false, false, nil
	}
	return v, true, false, p.ChangeVariantPrice(v.ID(), params.Price)
}

// draftDetails resolves d's unit, category, IVA and attributes. The
// category is created when it is missing and save is true.
func (s *Service) draftDetails(ctx context.Context, a Actor, c *draftCatalog, d ProductDraft, save bool) (domain.ProductDetails, error) {
	unit, err := c.unit(d.UnitCode)
	if err != nil {
		return domain.ProductDetails{}, err
	}
	rate, err := domain.ParseTaxRate(d.TaxRate)
	if err != nil {
		return domain.ProductDetails{}, err
	}
	attrs, err := domain.ResolveAttributes(c.attributes, d.Attributes)
	if err != nil {
		return domain.ProductDetails{}, err
	}
	details := domain.ProductDetails{
		Name: d.Name, Description: d.Description, BaseUnit: unit, TaxRate: rate, Attributes: attrs,
	}
	category, ok, err := s.draftCategory(ctx, a, c, d.Category, save)
	if ok {
		details.CategoryID = &category
	}
	return details, err
}

func boolCount(b bool) int {
	if b {
		return 1
	}
	return 0
}
