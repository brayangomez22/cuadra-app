package app_test

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

type fakeProducts struct{ *store }

// check enforces the per-tenant uniqueness of the variants' SKU and barcode
// against the other products (the product checks its own variants).
func (f fakeProducts) check(p *domain.Product) error {
	for _, other := range f.products {
		if other.TenantID() != p.TenantID() || other.ID() == p.ID() {
			continue
		}
		for _, v := range p.Variants() {
			for _, ov := range other.Variants() {
				if ov.SKU() == v.SKU() {
					return domain.ErrSKUTaken
				}
			}
		}
		for _, code := range barcodes(p) {
			if slices.Contains(barcodes(other), code) {
				return domain.ErrBarcodeTaken
			}
		}
	}
	if id := p.CategoryID(); id != nil {
		if c, ok := f.categories[*id]; !ok || c.TenantID() != p.TenantID() {
			return domain.ErrCategoryNotFound
		}
	}
	if u, ok := f.units[p.BaseUnit().ID]; !ok || u.TenantID() != p.TenantID() {
		return domain.ErrUnitNotFound
	}
	return nil
}

func (f fakeProducts) Create(_ context.Context, p *domain.Product) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.check(p); err != nil {
		return err
	}
	f.products[p.ID()] = cloneProduct(p)
	return nil
}

func (f fakeProducts) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.products[id]
	if !ok || p.TenantID() != tenantID {
		return nil, domain.ErrProductNotFound
	}
	return cloneProduct(p), nil
}

func (f fakeProducts) Update(_ context.Context, p *domain.Product) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.products[p.ID()]; !ok || old.TenantID() != p.TenantID() {
		return domain.ErrProductNotFound
	}
	if err := f.check(p); err != nil {
		return err
	}
	f.products[p.ID()] = cloneProduct(p)
	return nil
}

// Search matches names containing the query, case-insensitive.
func (f fakeProducts) Search(_ context.Context, tenantID uuid.UUID, filter domain.ProductFilter) (domain.ProductPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.searches = append(f.searches, filter)
	var matches []*domain.Product
	for _, p := range f.products {
		if p.TenantID() == tenantID && strings.Contains(strings.ToLower(p.Name()), strings.ToLower(filter.Query)) {
			matches = append(matches, cloneProduct(p))
		}
	}
	return domain.ProductPage{Items: matches, Total: len(matches)}, nil
}
