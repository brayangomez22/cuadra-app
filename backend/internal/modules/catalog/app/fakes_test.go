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
	attributes map[uuid.UUID]*domain.AttributeDefinition
	axes       map[uuid.UUID]*domain.VariantAxis
	// searches records the filters received by Search.
	searches []domain.ProductFilter
}

func cloneProduct(p *domain.Product) *domain.Product {
	variants := make([]domain.VariantSnapshot, 0, len(p.Variants()))
	for _, v := range p.Variants() {
		variants = append(variants, domain.VariantSnapshot{
			ID: v.ID(), SKU: v.SKU(), Barcode: v.Barcode(), Cost: v.Cost(), Price: v.Price(),
			AxisValues: v.AxisValues(), IsDefault: v.IsDefault(), Active: v.IsActive(), CreatedAt: v.CreatedAt(),
			Units: unitParams(v),
		})
	}
	c, err := domain.RehydrateProduct(domain.ProductSnapshot{
		ID: p.ID(), TenantID: p.TenantID(), Name: p.Name(),
		Description: p.Description(), CategoryID: p.CategoryID(), BaseUnit: p.BaseUnit(),
		TaxRate: p.TaxRate(), Attributes: p.Attributes(), Active: p.IsActive(), CreatedAt: p.CreatedAt(),
		Variants: variants,
	})
	if err != nil {
		panic(err)
	}
	return c
}

func unitParams(v *domain.ProductVariant) []domain.VariantUnitParams {
	var out []domain.VariantUnitParams
	for _, u := range v.Units() {
		out = append(out, domain.VariantUnitParams{Unit: u.Unit(), Factor: u.Factor(), Price: u.Price(), Barcode: u.Barcode()})
	}
	return out
}

// barcodes returns every barcode of p: its variants' and their units'.
func barcodes(p *domain.Product) []string {
	var out []string
	for _, v := range p.Variants() {
		if v.Barcode() != "" {
			out = append(out, v.Barcode())
		}
		for _, u := range v.Units() {
			if u.Barcode() != "" {
				out = append(out, u.Barcode())
			}
		}
	}
	return out
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
			attributes: map[uuid.UUID]*domain.AttributeDefinition{},
			axes:       map[uuid.UUID]*domain.VariantAxis{},
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
		Attributes:     fakeAttributes{e.store},
		VariantAxes:    fakeAxes{e.store},
		Seeds:          fakeSeeds{e.store},
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	e.meters = e.addUnit(t, e.actor.TenantID, "m", "Metro")
	// The axes the variant tests use, as a tenant would define them.
	e.addAxis(t, e.actor.TenantID, "talla", "S", "M", "L")
	e.addAxis(t, e.actor.TenantID, "color", "Rojo", "Azul")
	e.addAxis(t, e.actor.TenantID, "presentación", "Galón", "Cuarto")
	return e
}

// addAxis stores a variant axis of tenantID directly in the fake.
func (e *env) addAxis(t *testing.T, tenantID uuid.UUID, name string, values ...string) *domain.VariantAxis {
	t.Helper()
	a, err := domain.NewVariantAxis(tenantID, name, values, testNow)
	require.NoError(t, err)
	require.NoError(t, fakeAxes{e.store}.Create(t.Context(), a))
	return a
}

// addAttribute stores an attribute definition of tenantID directly in the fake.
func (e *env) addAttribute(t *testing.T, tenantID uuid.UUID, p domain.AttributeParams) *domain.AttributeDefinition {
	t.Helper()
	d, err := domain.NewAttributeDefinition(tenantID, p, testNow)
	require.NoError(t, err)
	require.NoError(t, fakeAttributes{e.store}.Create(t.Context(), d))
	return d
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
