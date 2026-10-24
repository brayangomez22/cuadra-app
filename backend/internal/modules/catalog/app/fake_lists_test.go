package app_test

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func cloneCategory(c *domain.Category) *domain.Category {
	out, err := domain.RehydrateCategory(domain.CategorySnapshot{
		ID: c.ID(), TenantID: c.TenantID(), Name: c.Name(), ParentID: c.ParentID(), CreatedAt: c.CreatedAt(),
	})
	if err != nil {
		panic(err)
	}
	return out
}

func cloneUnit(u *domain.UnitOfMeasure) *domain.UnitOfMeasure {
	out, err := domain.RehydrateUnit(domain.UnitSnapshot{
		ID: u.ID(), TenantID: u.TenantID(), Code: u.Code(), Name: u.Name(), CreatedAt: u.CreatedAt(),
	})
	if err != nil {
		panic(err)
	}
	return out
}

type fakeCategories struct{ *store }

func (f fakeCategories) checkParent(c *domain.Category) error {
	if id := c.ParentID(); id != nil {
		if parent, ok := f.categories[*id]; !ok || parent.TenantID() != c.TenantID() {
			return domain.ErrCategoryNotFound
		}
	}
	return nil
}

func (f fakeCategories) Create(_ context.Context, c *domain.Category) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.checkParent(c); err != nil {
		return err
	}
	f.categories[c.ID()] = cloneCategory(c)
	return nil
}

func (f fakeCategories) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Category, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.categories[id]
	if !ok || c.TenantID() != tenantID {
		return nil, domain.ErrCategoryNotFound
	}
	return cloneCategory(c), nil
}

func (f fakeCategories) List(_ context.Context, tenantID uuid.UUID) ([]*domain.Category, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.Category
	for _, c := range f.categories {
		if c.TenantID() == tenantID {
			out = append(out, cloneCategory(c))
		}
	}
	slices.SortFunc(out, func(a, b *domain.Category) int { return strings.Compare(a.Name(), b.Name()) })
	return out, nil
}

func (f fakeCategories) Update(_ context.Context, c *domain.Category) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.categories[c.ID()]; !ok || old.TenantID() != c.TenantID() {
		return domain.ErrCategoryNotFound
	}
	if err := f.checkParent(c); err != nil {
		return err
	}
	f.categories[c.ID()] = cloneCategory(c)
	return nil
}

func (f fakeCategories) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.categories[id]
	if !ok || c.TenantID() != tenantID {
		return domain.ErrCategoryNotFound
	}
	for _, other := range f.categories {
		if parent := other.ParentID(); parent != nil && *parent == id {
			return domain.ErrCategoryInUse
		}
	}
	for _, p := range f.products {
		if cat := p.CategoryID(); cat != nil && *cat == id {
			return domain.ErrCategoryInUse
		}
	}
	delete(f.categories, id)
	return nil
}

func (f fakeCategories) Ancestors(_ context.Context, tenantID, id uuid.UUID) ([]uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var chain []uuid.UUID
	for next := &id; next != nil; {
		c, ok := f.categories[*next]
		if !ok || c.TenantID() != tenantID {
			break
		}
		chain = append(chain, c.ID())
		next = c.ParentID()
	}
	return chain, nil
}

type fakeUnits struct{ *store }

// checkCode enforces the per-tenant uniqueness of the code.
func (f fakeUnits) checkCode(u *domain.UnitOfMeasure) error {
	for _, other := range f.units {
		if other.TenantID() == u.TenantID() && other.ID() != u.ID() && other.Code() == u.Code() {
			return domain.ErrUnitCodeTaken
		}
	}
	return nil
}

func (f fakeUnits) Create(_ context.Context, u *domain.UnitOfMeasure) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.checkCode(u); err != nil {
		return err
	}
	f.units[u.ID()] = cloneUnit(u)
	return nil
}

func (f fakeUnits) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.UnitOfMeasure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.units[id]
	if !ok || u.TenantID() != tenantID {
		return nil, domain.ErrUnitNotFound
	}
	return cloneUnit(u), nil
}

func (f fakeUnits) List(_ context.Context, tenantID uuid.UUID) ([]*domain.UnitOfMeasure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.UnitOfMeasure
	for _, u := range f.units {
		if u.TenantID() == tenantID {
			out = append(out, cloneUnit(u))
		}
	}
	slices.SortFunc(out, func(a, b *domain.UnitOfMeasure) int { return strings.Compare(a.Name(), b.Name()) })
	return out, nil
}

func (f fakeUnits) Update(_ context.Context, u *domain.UnitOfMeasure) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.units[u.ID()]; !ok || old.TenantID() != u.TenantID() {
		return domain.ErrUnitNotFound
	}
	if err := f.checkCode(u); err != nil {
		return err
	}
	f.units[u.ID()] = cloneUnit(u)
	return nil
}

func (f fakeUnits) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.units[id]
	if !ok || u.TenantID() != tenantID {
		return domain.ErrUnitNotFound
	}
	for _, p := range f.products {
		if p.BaseUnit().ID == id {
			return domain.ErrUnitInUse
		}
	}
	delete(f.units, id)
	return nil
}
