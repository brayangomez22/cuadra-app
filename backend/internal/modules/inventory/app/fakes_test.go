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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// In-memory fakes of the inventory ports. They copy entities in and out, like
// a database would, and enforce the same per-tenant constraints.

type levelKey struct{ tenantID, productID, locationID uuid.UUID }

type store struct {
	mu        sync.Mutex
	locations map[uuid.UUID]*domain.Location
	levels    map[levelKey]domain.StockLevelSnapshot
	movements []*domain.StockMovement
	policies  map[uuid.UUID]domain.StockPolicy
	// stockFilters and kardexFilters record the filters the queries received.
	stockFilters  []domain.StockFilter
	kardexFilters []domain.KardexFilter
}

func cloneLocation(l *domain.Location) *domain.Location {
	c, err := domain.RehydrateLocation(domain.LocationSnapshot{
		ID: l.ID(), TenantID: l.TenantID(), Name: l.Name(), Active: l.IsActive(), CreatedAt: l.CreatedAt(),
	})
	if err != nil {
		panic(err)
	}
	return c
}

func snapshot(l *domain.StockLevel) domain.StockLevelSnapshot {
	return domain.StockLevelSnapshot{
		TenantID: l.TenantID(), ProductID: l.ProductID(), LocationID: l.LocationID(),
		Quantity: l.Quantity(), AverageCost: l.AverageCost(),
	}
}

type fakeLocations struct{ *store }

func (f fakeLocations) Create(_ context.Context, l *domain.Location) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, other := range f.locations {
		if other.TenantID() == l.TenantID() && strings.EqualFold(other.Name(), l.Name()) {
			return domain.ErrLocationNameTaken
		}
	}
	f.locations[l.ID()] = cloneLocation(l)
	return nil
}

func (f fakeLocations) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Location, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.locations[id]
	if !ok || l.TenantID() != tenantID {
		return nil, domain.ErrLocationNotFound
	}
	return cloneLocation(l), nil
}

func (f fakeLocations) List(_ context.Context, tenantID uuid.UUID) ([]*domain.Location, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.Location
	for _, l := range f.locations {
		if l.TenantID() == tenantID {
			out = append(out, cloneLocation(l))
		}
	}
	slices.SortFunc(out, func(a, b *domain.Location) int { return strings.Compare(a.Name(), b.Name()) })
	return out, nil
}

type fakePolicies struct{ *store }

func (f fakePolicies) Get(_ context.Context, tenantID uuid.UUID) (domain.StockPolicy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.policies[tenantID], nil
}

type fakeStock struct{ *store }

func (f fakeStock) Apply(_ context.Context, tenantID, productID uuid.UUID, locationIDs []uuid.UUID, change domain.StockChange) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	levels := make([]*domain.StockLevel, 0, len(locationIDs))
	for _, locationID := range locationIDs {
		// The database's foreign key: the location must exist in the tenant.
		if l, ok := f.locations[locationID]; !ok || l.TenantID() != tenantID {
			return domain.ErrLocationNotFound
		}
		s, ok := f.levels[levelKey{tenantID, productID, locationID}]
		if !ok {
			s = domain.StockLevelSnapshot{TenantID: tenantID, ProductID: productID, LocationID: locationID}
		}
		level, err := domain.RehydrateStockLevel(s)
		if err != nil {
			return err
		}
		levels = append(levels, level)
	}
	movements, err := change(levels)
	if err != nil {
		return err
	}
	for _, l := range levels {
		f.levels[levelKey{tenantID, productID, l.LocationID()}] = snapshot(l)
	}
	f.movements = append(f.movements, movements...)
	return nil
}

func (f fakeStock) ListLevels(_ context.Context, tenantID uuid.UUID, filter domain.StockFilter) (domain.StockLevelPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stockFilters = append(f.stockFilters, filter)
	var page domain.StockLevelPage
	for k, s := range f.levels {
		if k.tenantID != tenantID || k.locationID != filter.LocationID || filter.ProductID != nil && k.productID != *filter.ProductID {
			continue
		}
		l, err := domain.RehydrateStockLevel(s)
		if err != nil {
			return domain.StockLevelPage{}, err
		}
		page.Items = append(page.Items, l)
	}
	page.Total = len(page.Items)
	return page, nil
}

func (f fakeStock) Kardex(_ context.Context, tenantID uuid.UUID, filter domain.KardexFilter) (domain.MovementPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kardexFilters = append(f.kardexFilters, filter)
	var page domain.MovementPage
	for _, m := range slices.Backward(f.movements) {
		if m.TenantID() != tenantID || m.ProductID() != filter.ProductID ||
			filter.LocationID != nil && m.LocationID() != *filter.LocationID {
			continue
		}
		page.Items = append(page.Items, m)
	}
	page.Total = len(page.Items)
	return page, nil
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
		store: &store{
			locations: map[uuid.UUID]*domain.Location{},
			levels:    map[levelKey]domain.StockLevelSnapshot{},
			policies:  map[uuid.UUID]domain.StockPolicy{},
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
		Locations:      fakeLocations{e.store},
		Stock:          fakeStock{e.store},
		Policies:       fakePolicies{e.store},
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// level returns the stored level of a product at a location, or an empty one.
func (e *env) level(productID, locationID uuid.UUID) domain.StockLevelSnapshot {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	return e.store.levels[levelKey{e.actor.TenantID, productID, locationID}]
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
