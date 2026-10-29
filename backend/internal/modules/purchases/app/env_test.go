package app_test

import (
	"context"
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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc       *app.Service
	suppliers *fakeSuppliers
	purchases *fakePurchases
	catalog   *fakeCatalog
	stock     *fakeStock
	actor     app.Actor
	location  uuid.UUID
	spans     *tracetest.SpanRecorder
	metrics   *sdkmetric.ManualReader
}

// testNow is 2026-10-28 at 21:00 in Bogotá, already the 29th in UTC.
var testNow = time.Date(2026, 10, 29, 2, 0, 0, 0, time.UTC)

// today is testNow's day in Bogotá.
var today = domain.NewDate(2026, time.October, 28)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		suppliers: newFakeSuppliers(), purchases: &fakePurchases{},
		catalog: &fakeCatalog{variants: map[uuid.UUID]app.VariantInfo{}}, stock: &fakeStock{},
		actor:    app.Actor{TenantID: newID(), UserID: newID()},
		location: newID(),
		spans:    tracetest.NewSpanRecorder(), metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Suppliers: e.suppliers, Purchases: e.purchases, Catalog: e.catalog, Stock: e.stock,
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

func supplierParams(name, nit string, days int) domain.SupplierParams {
	return domain.SupplierParams{Name: name, NIT: nit, PaymentTermDays: days, Active: true}
}

// addSupplier creates an active supplier of the actor's tenant.
func (e *env) addSupplier(t *testing.T, name, nit string, days int) *domain.Supplier {
	t.Helper()
	s, err := e.svc.CreateSupplier(t.Context(), e.actor, supplierParams(name, nit, days))
	require.NoError(t, err)
	return s.Supplier
}

// variant registers an active variant in the catalog, with alternate units.
func (e *env) variant(name, label, baseUnit, taxRate string, units ...app.UnitInfo) uuid.UUID {
	id := newID()
	e.catalog.variants[id] = app.VariantInfo{
		ID: id, ProductName: name, Label: label, BaseUnitCode: baseUnit, TaxRate: dec(taxRate), Active: true, Units: units,
	}
	return id
}

func unit(code, factor string) app.UnitInfo {
	return app.UnitInfo{UnitID: newID(), Code: code, Factor: dec(factor)}
}

// purchase is a purchase of the supplier invoiced two days ago.
func (e *env) purchase(supplier uuid.UUID, invoice string, pay domain.Payment, lines ...app.PurchaseLineInput) app.PurchaseInput {
	return app.PurchaseInput{
		SupplierID: supplier, LocationID: e.location, InvoiceNumber: invoice, IssuedOn: today.AddDays(-2),
		Payment: pay, Lines: lines,
	}
}

func line(variant uuid.UUID, u *app.UnitInfo, qty, cost string) app.PurchaseLineInput {
	l := app.PurchaseLineInput{VariantID: variant, Quantity: dec(qty), UnitCost: dec(cost)}
	if u != nil {
		l.UnitID = &u.UnitID
	}
	return l
}

func (e *env) span(t *testing.T, name string) map[attribute.Key]attribute.Value {
	t.Helper()
	var found []sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			found = append(found, s)
		}
	}
	require.NotEmpty(t, found, "spans named %s", name)
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range found[len(found)-1].Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return attrs
}

// counter returns the sum of the data points of an int64 counter whose
// attribute key equals value ("" sums every point).
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
				if v, ok := dp.Attributes.Value(attribute.Key(key)); key == "" || (ok && v.AsString() == value) {
					total += dp.Value
				}
			}
		}
	}
	return total
}
