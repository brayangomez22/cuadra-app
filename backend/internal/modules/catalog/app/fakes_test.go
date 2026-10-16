package app_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// In-memory fakes of the catalog ports. They copy entities in and out, like
// a database would, and enforce the same per-tenant constraints.

type store struct {
	mu         sync.Mutex
	products   map[uuid.UUID]*domain.Product
	categories map[uuid.UUID]*domain.Category
	units      map[uuid.UUID]*domain.UnitOfMeasure
	// searches records the filters received by Search.
	searches []domain.ProductFilter
}

func cloneProduct(p *domain.Product) *domain.Product {
	variants := make([]domain.VariantSnapshot, 0, len(p.Variants()))
	for _, v := range p.Variants() {
		variants = append(variants, domain.VariantSnapshot{
			ID: v.ID(), SKU: v.SKU(), Barcode: v.Barcode(), Cost: v.Cost(), Price: v.Price(),
			AxisValues: v.AxisValues(), IsDefault: v.IsDefault(), Active: v.IsActive(), CreatedAt: v.CreatedAt(),
		})
	}
	c, err := domain.RehydrateProduct(domain.ProductSnapshot{
		ID: p.ID(), TenantID: p.TenantID(), Name: p.Name(),
		Description: p.Description(), CategoryID: p.CategoryID(), BaseUnit: p.BaseUnit(),
		TaxRate: p.TaxRate(), Active: p.IsActive(), CreatedAt: p.CreatedAt(), Variants: variants,
	})
	if err != nil {
		panic(err)
	}
	return c
}

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
				if v.Barcode() != "" && ov.Barcode() == v.Barcode() {
					return domain.ErrBarcodeTaken
				}
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

func (f fakeUnits) CreateMany(_ context.Context, tenantID uuid.UUID, units []*domain.UnitOfMeasure) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored := map[uuid.UUID]*domain.UnitOfMeasure{}
	for _, u := range units {
		if u.TenantID() != tenantID {
			return errors.New("fake: unit of another tenant")
		}
		if err := f.checkCode(u); err != nil {
			return err
		}
		for _, other := range stored {
			if other.Code() == u.Code() {
				return domain.ErrUnitCodeTaken
			}
		}
		stored[u.ID()] = cloneUnit(u)
	}
	maps.Copy(f.units, stored)
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

func (f fakeUnits) GetByCode(_ context.Context, tenantID uuid.UUID, code string) (*domain.UnitOfMeasure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.units {
		if u.TenantID() == tenantID && u.Code() == code {
			return cloneUnit(u), nil
		}
	}
	return nil, domain.ErrUnitNotFound
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

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc   *app.Service
	store *store
	actor app.Actor
	// meters is a unit of the actor's tenant.
	meters  *domain.UnitOfMeasure
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

var testNow = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		store: &store{
			products:   map[uuid.UUID]*domain.Product{},
			categories: map[uuid.UUID]*domain.Category{},
			units:      map[uuid.UUID]*domain.UnitOfMeasure{},
		},
		actor: app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())},
		spans: tracetest.NewSpanRecorder(), metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Products:       fakeProducts{e.store},
		Categories:     fakeCategories{e.store},
		Units:          fakeUnits{e.store},
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	e.meters = e.addUnit(t, e.actor.TenantID, "m", "Metro")
	return e
}

// addUnit stores a unit of tenantID directly in the fake, without the
// service.
func (e *env) addUnit(t *testing.T, tenantID uuid.UUID, code, name string) *domain.UnitOfMeasure {
	t.Helper()
	u, err := domain.NewUnit(tenantID, code, name, testNow)
	require.NoError(t, err)
	require.NoError(t, fakeUnits{e.store}.Create(t.Context(), u))
	return u
}

// span returns the attributes of the only ended span named name.
func (e *env) span(t *testing.T, name string) (sdktrace.ReadOnlySpan, map[attribute.Key]attribute.Value) {
	t.Helper()
	var found []sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			found = append(found, s)
		}
	}
	require.Len(t, found, 1, "spans named %s", name)
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range found[0].Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return found[0], attrs
}

// counter returns the sum of the data points of an int64 counter whose
// attribute key equals value (any data point when key is "").
func (e *env) counter(t *testing.T, name, key, value string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "%s is not an int64 counter", name)
			for _, dp := range sum.DataPoints {
				if v, ok := dp.Attributes.Value(attribute.Key(key)); key == "" || ok && v.AsString() == value {
					total += dp.Value
				}
			}
		}
	}
	return total
}
