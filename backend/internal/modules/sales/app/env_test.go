package app_test

import (
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

var testNow = time.Date(2026, 10, 10, 9, 30, 0, 0, time.FixedZone("COT", -5*3600))

type env struct {
	svc     *app.Service
	store   *store
	actor   app.Actor
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

// newEnv builds a Service on the fakes; rules go before the base price rule.
func newEnv(t *testing.T, rules ...app.PriceRule) *env {
	t.Helper()
	e := &env{
		store: &store{
			variants:  map[catalogKey]app.VariantInfo{},
			policies:  map[uuid.UUID]domain.SalesPolicy{},
			sales:     map[uuid.UUID]*domain.Sale{},
			numbers:   map[uuid.UUID]int64{},
			sessions:  map[uuid.UUID]*domain.CashSession{},
			customers: map[catalogKey]bool{},
		},
		actor:   app.Actor{TenantID: newID(), UserID: newID()},
		spans:   tracetest.NewSpanRecorder(),
		metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	chain, err := app.NewPriceChain(rules...)
	require.NoError(t, err)
	svc, err := app.NewService(app.Deps{
		Catalog:        fakeCatalog{e.store},
		Prices:         recordingPrices{e.store, chain},
		Stock:          fakeStock{e.store},
		Sales:          fakeSales{e.store},
		Policies:       fakePolicies{e.store},
		CashSessions:   fakeCash{e.store},
		Customers:      fakeCustomers{e.store},
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	// The sales tests predate the cash sessions: their tenant sells without
	// one. The cash tests require it with requireCashSession.
	e.store.policies[e.actor.TenantID] = domain.SalesPolicy{CashSessionOptional: true}
	return e
}

// addCustomer puts a customer in the actor's tenant and returns its id.
func (e *env) addCustomer(active bool) uuid.UUID {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	id := newID()
	e.store.customers[catalogKey{e.actor.TenantID, id}] = active
	return id
}

// requireCashSession makes the tenant require an open cash session to sell,
// the default of a tenant without settings.
func (e *env) requireCashSession() {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	p := e.store.policies[e.actor.TenantID]
	p.CashSessionOptional = false
	e.store.policies[e.actor.TenantID] = p
}

// addProduct puts a simple product in the actor's catalog and returns its
// default variant, whose id is not the product's.
func (e *env) addProduct(name, price, rate string, active bool) uuid.UUID {
	productID, variantID := newID(), newID()
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	e.store.variants[catalogKey{e.actor.TenantID, variantID}] = app.VariantInfo{
		ID: variantID, ProductID: productID, ProductName: name, Price: dec(price), TaxRate: dec(rate), Active: active,
	}
	return variantID
}

// addVariant adds a variant with label to the product of variant sibling
// and returns it.
func (e *env) addVariant(sibling uuid.UUID, label, price string, active bool) uuid.UUID {
	id := newID()
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	v := e.store.variants[catalogKey{e.actor.TenantID, sibling}]
	v.ID, v.Label, v.Price, v.Active = id, label, dec(price), active
	e.store.variants[catalogKey{e.actor.TenantID, id}] = v
	return id
}

// productOf returns the product of a variant of the actor's catalog.
func (e *env) productOf(variantID uuid.UUID) uuid.UUID {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	return e.store.variants[catalogKey{e.actor.TenantID, variantID}].ProductID
}

// setMaxDiscount sets the tenant's discount limit for users without
// sales:discount.
func (e *env) setMaxDiscount(t *testing.T, fraction string) {
	t.Helper()
	limit, err := domain.NewDiscountLimit(dec(fraction))
	require.NoError(t, err)
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	p := e.store.policies[e.actor.TenantID]
	p.MaxDiscount = limit
	e.store.policies[e.actor.TenantID] = p
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
