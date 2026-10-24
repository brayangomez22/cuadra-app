package app_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func cloneAttribute(d *domain.AttributeDefinition) *domain.AttributeDefinition {
	out, err := domain.RehydrateAttributeDefinition(domain.AttributeSnapshot{
		ID: d.ID(), TenantID: d.TenantID(), Key: d.Key(), Label: d.Label(), Type: d.Type(),
		Options: d.Options(), Required: d.Required(), CreatedAt: d.CreatedAt(),
	})
	if err != nil {
		panic(err)
	}
	return out
}

type fakeAttributes struct{ *store }

func (f fakeAttributes) Create(_ context.Context, d *domain.AttributeDefinition) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, other := range f.attributes {
		if other.TenantID() == d.TenantID() && other.Key() == d.Key() {
			return domain.ErrAttributeKeyTaken
		}
	}
	f.attributes[d.ID()] = cloneAttribute(d)
	return nil
}

func (f fakeAttributes) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.AttributeDefinition, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.attributes[id]
	if !ok || d.TenantID() != tenantID {
		return nil, domain.ErrAttributeNotFound
	}
	return cloneAttribute(d), nil
}

func (f fakeAttributes) List(_ context.Context, tenantID uuid.UUID) ([]*domain.AttributeDefinition, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.AttributeDefinition
	for _, d := range f.attributes {
		if d.TenantID() == tenantID {
			out = append(out, cloneAttribute(d))
		}
	}
	slices.SortFunc(out, func(a, b *domain.AttributeDefinition) int { return strings.Compare(a.Label(), b.Label()) })
	return out, nil
}

// valuesOf returns the values the tenant's products have for key.
func (f fakeAttributes) valuesOf(tenantID uuid.UUID, key string) []domain.AttributeValue {
	var out []domain.AttributeValue
	for _, p := range f.products {
		if v, ok := p.Attributes().Values()[key]; ok && p.TenantID() == tenantID {
			out = append(out, v)
		}
	}
	return out
}

func (f fakeAttributes) Update(_ context.Context, d *domain.AttributeDefinition, removed []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.attributes[d.ID()]; !ok || old.TenantID() != d.TenantID() {
		return domain.ErrAttributeNotFound
	}
	for _, v := range f.valuesOf(d.TenantID(), d.Key()) {
		if slices.Contains(removed, v.Text()) {
			return domain.ErrAttributeOptionInUse
		}
	}
	f.attributes[d.ID()] = cloneAttribute(d)
	return nil
}

func (f fakeAttributes) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.attributes[id]
	if !ok || d.TenantID() != tenantID {
		return domain.ErrAttributeNotFound
	}
	if len(f.valuesOf(tenantID, d.Key())) > 0 {
		return domain.ErrAttributeInUse
	}
	delete(f.attributes, id)
	return nil
}

func cloneAxis(a *domain.VariantAxis) *domain.VariantAxis {
	out, err := domain.RehydrateVariantAxis(domain.VariantAxisSnapshot{
		ID: a.ID(), TenantID: a.TenantID(), Name: a.Name(), Values: a.Values(), CreatedAt: a.CreatedAt(),
	})
	if err != nil {
		panic(err)
	}
	return out
}

type fakeAxes struct{ *store }

func (f fakeAxes) Create(_ context.Context, a *domain.VariantAxis) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, other := range f.axes {
		if other.TenantID() == a.TenantID() && other.Name() == a.Name() {
			return domain.ErrAxisNameTaken
		}
	}
	f.axes[a.ID()] = cloneAxis(a)
	return nil
}

func (f fakeAxes) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.VariantAxis, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.axes[id]
	if !ok || a.TenantID() != tenantID {
		return nil, domain.ErrAxisNotFound
	}
	return cloneAxis(a), nil
}

func (f fakeAxes) List(_ context.Context, tenantID uuid.UUID) ([]*domain.VariantAxis, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.VariantAxis
	for _, a := range f.axes {
		if a.TenantID() == tenantID {
			out = append(out, cloneAxis(a))
		}
	}
	slices.SortFunc(out, func(a, b *domain.VariantAxis) int { return strings.Compare(a.Name(), b.Name()) })
	return out, nil
}

// valuesOf returns the values the tenant's variants have on axis name.
func (f fakeAxes) valuesOf(tenantID uuid.UUID, name string) []string {
	var out []string
	for _, p := range f.products {
		if p.TenantID() != tenantID {
			continue
		}
		for _, v := range p.Variants() {
			if value, ok := v.AxisValues()[name]; ok {
				out = append(out, value)
			}
		}
	}
	return out
}

func (f fakeAxes) Update(_ context.Context, a *domain.VariantAxis, removed []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.axes[a.ID()]; !ok || old.TenantID() != a.TenantID() {
		return domain.ErrAxisNotFound
	}
	for _, v := range f.valuesOf(a.TenantID(), a.Name()) {
		if slices.ContainsFunc(removed, func(r string) bool { return strings.EqualFold(r, v) }) {
			return domain.ErrAxisValueInUse
		}
	}
	f.axes[a.ID()] = cloneAxis(a)
	return nil
}

func (f fakeAxes) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.axes[id]
	if !ok || a.TenantID() != tenantID {
		return domain.ErrAxisNotFound
	}
	if len(f.valuesOf(tenantID, a.Name())) > 0 {
		return domain.ErrAxisInUse
	}
	delete(f.axes, id)
	return nil
}

// fakeSeeds stores a seed with the other fakes, all or nothing.
type fakeSeeds struct{ *store }

func (f fakeSeeds) Seed(ctx context.Context, tenantID uuid.UUID, seed domain.TenantSeed) (err error) {
	f.mu.Lock()
	units, categories := maps.Clone(f.units), maps.Clone(f.categories)
	attributes, axes := maps.Clone(f.attributes), maps.Clone(f.axes)
	f.mu.Unlock()
	defer func() {
		if err != nil {
			f.mu.Lock()
			f.units, f.categories, f.attributes, f.axes = units, categories, attributes, axes
			f.mu.Unlock()
		}
	}()

	for _, u := range seed.Units {
		if u.TenantID() != tenantID {
			return errors.New("fake: unit of another tenant")
		}
		if err := fakeUnits(f).Create(ctx, u); err != nil {
			return err
		}
	}
	for _, c := range seed.Categories {
		if err := fakeCategories(f).Create(ctx, c); err != nil {
			return err
		}
	}
	for _, d := range seed.Attributes {
		if err := fakeAttributes(f).Create(ctx, d); err != nil {
			return err
		}
	}
	for _, a := range seed.VariantAxes {
		if err := fakeAxes(f).Create(ctx, a); err != nil {
			return err
		}
	}
	return nil
}
