package app

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// draftCatalog is what the tenant's catalog has for a batch of drafts,
// read once: its units, categories, attributes and axes, and the products
// that have the drafts' SKUs.
type draftCatalog struct {
	units      []*domain.UnitOfMeasure
	categories []*domain.Category
	attributes []*domain.AttributeDefinition
	axes       []*domain.VariantAxis
	// bySKU maps each SKU (as stored) to its product.
	bySKU map[string]*domain.Product
}

func (s *Service) loadDraftCatalog(ctx context.Context, a Actor, drafts []ProductDraft) (*draftCatalog, error) {
	c := &draftCatalog{bySKU: map[string]*domain.Product{}}
	var err error
	if c.units, err = s.Units.List(ctx, a.TenantID); err != nil {
		return nil, err
	}
	if c.categories, err = s.Categories.List(ctx, a.TenantID); err != nil {
		return nil, err
	}
	if c.attributes, err = s.Attributes.List(ctx, a.TenantID); err != nil {
		return nil, err
	}
	if c.axes, err = s.VariantAxes.List(ctx, a.TenantID); err != nil {
		return nil, err
	}
	var skus []string
	for _, d := range drafts {
		for _, v := range d.Variants {
			skus = append(skus, storedSKU(v.SKU))
		}
	}
	products, err := s.Products.BySKU(ctx, a.TenantID, skus)
	if err != nil {
		return nil, err
	}
	for _, p := range products {
		c.remember(p)
	}
	return c, nil
}

// remember makes p's SKUs known to the next drafts of the batch.
func (c *draftCatalog) remember(p *domain.Product) {
	for _, v := range p.Variants() {
		c.bySKU[v.SKU()] = p
	}
}

// existing returns the product that has d's SKUs, or nil if none does;
// domain.ErrVariantsOfSeveralProducts if they are in different products.
func (c *draftCatalog) existing(d ProductDraft) (*domain.Product, error) {
	var found *domain.Product
	for _, v := range d.Variants {
		p, ok := c.bySKU[storedSKU(v.SKU)]
		switch {
		case !ok:
		case found != nil && found.ID() != p.ID():
			return nil, domain.ErrVariantsOfSeveralProducts
		default:
			found = p
		}
	}
	return found, nil
}

// unit returns the tenant's unit with code, without regard to case;
// domain.ErrUnitNotFound if there is none.
func (c *draftCatalog) unit(code string) (domain.UnitRef, error) {
	for _, u := range c.units {
		if strings.EqualFold(u.Code(), strings.TrimSpace(code)) {
			return u.Ref(), nil
		}
	}
	return domain.UnitRef{}, domain.ErrUnitNotFound
}

// draftCategory returns the id of the tenant's category named name
// (without regard to case); ok is false for no name. A missing one is
// checked and, when save is true, created at the top level.
func (s *Service) draftCategory(ctx context.Context, a Actor, c *draftCatalog, name string, save bool) (_ uuid.UUID, ok bool, _ error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, false, nil
	}
	for _, cat := range c.categories {
		if strings.EqualFold(cat.Name(), name) {
			return cat.ID(), true, nil
		}
	}
	cat, err := domain.NewCategory(a.TenantID, name, nil, s.Now())
	if err != nil || !save {
		return uuid.Nil, false, err
	}
	if err := s.Categories.Create(ctx, cat); err != nil {
		return uuid.Nil, false, err
	}
	c.categories = append(c.categories, cat)
	return cat.ID(), true, nil
}

// variantWithSKU returns p's variant with sku, compared as stored.
func variantWithSKU(p *domain.Product, sku string) (*domain.ProductVariant, bool) {
	for _, v := range p.Variants() {
		if v.SKU() == storedSKU(sku) {
			return v, true
		}
	}
	return nil, false
}

// storedSKU is sku as the catalog stores it.
func storedSKU(sku string) string { return strings.ToUpper(strings.TrimSpace(sku)) }
