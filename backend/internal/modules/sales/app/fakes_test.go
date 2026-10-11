package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// In-memory fakes of the sales ports.

type productKey struct{ tenantID, productID uuid.UUID }

type reservation struct {
	tenantID, userID, locationID, saleID uuid.UUID
	lines                                []app.StockLine
}

type store struct {
	mu       sync.Mutex
	products map[productKey]app.ProductInfo
	policies map[uuid.UUID]domain.SalesPolicy
	sales    map[uuid.UUID]*domain.Sale
	reserved []reservation
	// reserveErr, when set, makes the next reservations fail.
	reserveErr error
	// reversed records the sales whose stock was given back; reverseErr,
	// when set, makes the next reversals fail.
	reversed   []uuid.UUID
	reverseErr error
	// numbers is the last number given at each location.
	numbers map[uuid.UUID]int64
	// filters records the filters List received.
	filters []domain.SaleFilter
	// catalogCalls records the ids each catalog lookup received.
	catalogCalls [][]uuid.UUID
}

type fakeCatalog struct{ *store }

func (f fakeCatalog) Products(_ context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]app.ProductInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.catalogCalls = append(f.catalogCalls, ids)
	var out []app.ProductInfo
	for _, id := range ids {
		if p, ok := f.products[productKey{tenantID, id}]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

type fakeStock struct{ *store }

func (f fakeStock) ReserveSale(_ context.Context, tenantID, userID, locationID, saleID uuid.UUID, lines []app.StockLine) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reserveErr != nil {
		return f.reserveErr
	}
	f.reserved = append(f.reserved, reservation{tenantID, userID, locationID, saleID, lines})
	return nil
}

func (f fakeStock) ReverseSale(_ context.Context, _, _, saleID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reverseErr != nil {
		return f.reverseErr
	}
	f.reversed = append(f.reversed, saleID)
	return nil
}

// clone copies a sale in or out of the fake, as a database would.
func clone(s *domain.Sale) *domain.Sale {
	c, err := domain.RehydrateSale(s.Snapshot())
	if err != nil {
		panic(err)
	}
	return c
}

// fakeSales stores the sale only if reserve succeeds, as one transaction
// would, and numbers it per location.
type fakeSales struct{ *store }

func (f fakeSales) SaveConfirmed(ctx context.Context, s *domain.Sale, reserve func(context.Context) error) error {
	if err := reserve(ctx); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := s.AssignNumber(f.numbers[s.LocationID()] + 1); err != nil {
		return err
	}
	f.numbers[s.LocationID()]++
	f.sales[s.ID()] = clone(s)
	return nil
}

func (f fakeSales) Get(_ context.Context, tenantID, id uuid.UUID) (*domain.Sale, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sales[id]
	if !ok || s.TenantID() != tenantID {
		return nil, domain.ErrSaleNotFound
	}
	return clone(s), nil
}

func (f fakeSales) List(_ context.Context, tenantID uuid.UUID, filter domain.SaleFilter) (domain.SalePage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filters = append(f.filters, filter)
	var page domain.SalePage
	for _, s := range f.sales {
		at := *s.ConfirmedAt()
		if s.TenantID() != tenantID || at.Before(filter.From) || !at.Before(filter.To) ||
			filter.LocationID != nil && s.LocationID() != *filter.LocationID {
			continue
		}
		page.Items = append(page.Items, domain.SummaryOf(s))
	}
	page.Total = len(page.Items)
	return page, nil
}

// Void changes a copy and stores it only if change succeeds, as one
// transaction would.
func (f fakeSales) Void(ctx context.Context, tenantID, id uuid.UUID, change func(context.Context, *domain.Sale) error) (*domain.Sale, error) {
	s, err := f.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := change(ctx, s); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sales[id] = clone(s)
	return s, nil
}

type fakePolicies struct{ *store }

func (f fakePolicies) Get(_ context.Context, tenantID uuid.UUID) (domain.SalesPolicy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.policies[tenantID], nil
}

var testNow = time.Date(2026, 10, 10, 9, 30, 0, 0, time.FixedZone("COT", -5*3600))

type env struct {
	svc     *app.Service
	store   *store
	actor   app.Actor
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		store: &store{
			products: map[productKey]app.ProductInfo{},
			policies: map[uuid.UUID]domain.SalesPolicy{},
			sales:    map[uuid.UUID]*domain.Sale{},
			numbers:  map[uuid.UUID]int64{},
		},
		actor:   app.Actor{TenantID: newID(), UserID: newID()},
		spans:   tracetest.NewSpanRecorder(),
		metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	svc, err := app.NewService(app.Deps{
		Products:       fakeCatalog{e.store},
		Stock:          fakeStock{e.store},
		Sales:          fakeSales{e.store},
		Policies:       fakePolicies{e.store},
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// addProduct puts a product in the actor's catalog.
func (e *env) addProduct(name, price, rate string, active bool) uuid.UUID {
	id := newID()
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	e.store.products[productKey{e.actor.TenantID, id}] = app.ProductInfo{
		ID: id, Name: name, Price: dec(price), TaxRate: dec(rate), Active: active,
	}
	return id
}

// setMaxDiscount sets the tenant's discount limit for users without
// sales:discount.
func (e *env) setMaxDiscount(t *testing.T, fraction string) {
	t.Helper()
	limit, err := domain.NewDiscountLimit(dec(fraction))
	require.NoError(t, err)
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	e.store.policies[e.actor.TenantID] = domain.SalesPolicy{MaxDiscount: limit}
}

func (e *env) storedSales() int {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	return len(e.store.sales)
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

func (e *env) collect(t *testing.T) []metricdata.Metrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var out []metricdata.Metrics
	for _, sm := range rm.ScopeMetrics {
		out = append(out, sm.Metrics...)
	}
	return out
}

// counter returns the sum of the data points of an int64 counter.
func (e *env) counter(t *testing.T, name string) int64 {
	t.Helper()
	var total int64
	for _, m := range e.collect(t) {
		if m.Name != name {
			continue
		}
		sum, ok := m.Data.(metricdata.Sum[int64])
		require.True(t, ok, "%s is not an int64 counter", name)
		for _, dp := range sum.DataPoints {
			total += dp.Value
		}
	}
	return total
}

// histogram returns the count and sum of a float64 histogram.
func (e *env) histogram(t *testing.T, name string) (count uint64, sum float64) {
	t.Helper()
	for _, m := range e.collect(t) {
		if m.Name != name {
			continue
		}
		h, ok := m.Data.(metricdata.Histogram[float64])
		require.True(t, ok, "%s is not a float64 histogram", name)
		for _, dp := range h.DataPoints {
			count += dp.Count
			sum += dp.Sum
		}
	}
	return count, sum
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }
