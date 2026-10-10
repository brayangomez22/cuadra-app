package app_test

import (
	"context"
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
	// searches records the filters received by Search.
	searches []domain.ProductFilter
}

func cloneProduct(p *domain.Product) *domain.Product {
	c, err := domain.RehydrateProduct(domain.ProductSnapshot{
		ID: p.ID(), TenantID: p.TenantID(), SKU: p.SKU(), Barcode: p.Barcode(), Name: p.Name(),
		Description: p.Description(), CategoryID: p.CategoryID(), BaseUnit: p.BaseUnit(),
		Cost: p.Cost(), Price: p.Price(), TaxRate: p.TaxRate(), Active: p.IsActive(), CreatedAt: p.CreatedAt(),
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

type fakeProducts struct{ *store }

func (f fakeProducts) check(p *domain.Product) error {
	for _, other := range f.products {
		if other.TenantID() != p.TenantID() || other.ID() == p.ID() {
			continue
		}
		if other.SKU() == p.SKU() {
			return domain.ErrSKUTaken
		}
		if p.Barcode() != "" && other.Barcode() == p.Barcode() {
			return domain.ErrBarcodeTaken
		}
	}
	if id := p.CategoryID(); id != nil {
		if c, ok := f.categories[*id]; !ok || c.TenantID() != p.TenantID() {
			return domain.ErrCategoryNotFound
		}
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

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc     *app.Service
	store   *store
	actor   app.Actor
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

var testNow = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		store: &store{products: map[uuid.UUID]*domain.Product{}, categories: map[uuid.UUID]*domain.Category{}},
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
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
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
