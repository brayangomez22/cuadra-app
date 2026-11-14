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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc      *app.Service
	store    *store
	invoicer *fakeInvoicer
	sales    fakeSales
	buyers   fakeBuyers
	actor    app.Actor
	now      time.Time
	spans    *tracetest.SpanRecorder
	metrics  *sdkmetric.ManualReader
}

// testNow is 2026-03-03 at 21:00 in Bogotá, already the 4th in UTC.
var testNow = time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		store:    newStore(),
		invoicer: newInvoicer(),
		sales:    fakeSales{sales: map[uuid.UUID]app.SaleData{}},
		buyers:   fakeBuyers{buyers: map[uuid.UUID]domain.Buyer{}},
		actor:    app.Actor{TenantID: newID(), UserID: newID()},
		now:      testNow,
		spans:    tracetest.NewSpanRecorder(), metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Settings:       fakeSettings{e.store},
		Resolutions:    fakeResolutions{e.store},
		Documents:      fakeDocuments{e.store},
		Invoicer:       e.invoicer,
		Sales:          e.sales,
		Buyers:         e.buyers,
		Now:            func() time.Time { return e.now },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// enable turns emission on for the actor's tenant, with defaultKind.
func (e *env) enable(t *testing.T, defaultKind domain.Kind) {
	t.Helper()
	_, err := e.svc.UpdateSettings(t.Context(), e.actor, true, string(defaultKind))
	require.NoError(t, err)
}

// resolution registers a resolution of kind with prefix and range, valid
// all of 2026.
func (e *env) resolution(t *testing.T, kind domain.Kind, prefix string, from, to int64) uuid.UUID {
	t.Helper()
	v, err := e.svc.CreateResolution(t.Context(), e.actor, domain.ResolutionParams{
		Kind: kind, Prefix: prefix, Number: "18764000001234", From: from, To: to,
		ValidFrom: domain.NewDate(2026, time.January, 1), ValidTo: domain.NewDate(2026, time.December, 31),
	})
	require.NoError(t, err)
	return v.Resolution.ID()
}

// sale stores a sale of two lines without customer and returns its id.
func (e *env) sale(customerID *uuid.UUID) uuid.UUID {
	id := newID()
	e.sales.sales[id] = app.SaleData{
		Number: 7, CustomerID: customerID,
		Lines: []app.SaleLine{
			{VariantID: newID(), Name: "Producto A", Quantity: dec("2.5"), UnitPrice: dec("4000"), TaxRate: dec("0.19"), Tax: dec("1900"), Total: dec("11900")},
			{VariantID: newID(), Name: "Producto B", Quantity: dec("1"), UnitPrice: dec("10000"), TaxRate: dec("0"), Total: dec("10000")},
		},
		Payments: []app.SalePayment{{Method: "cash", Amount: dec("21900")}},
		Subtotal: dec("20000"), Tax: dec("1900"), Total: dec("21900"),
	}
	return id
}

// register gives a sale without customer its document.
func (e *env) register(t *testing.T, saleID uuid.UUID) *domain.Document {
	t.Helper()
	require.NoError(t, e.svc.RegisterSale(t.Context(), e.actor, app.SaleDocumentRequest{SaleID: saleID}))
	return e.documentOf(t, saleID)
}

// documentOf returns the stored document of a sale.
func (e *env) documentOf(t *testing.T, saleID uuid.UUID) *domain.Document {
	t.Helper()
	page, err := e.svc.ListDocuments(t.Context(), e.actor, domain.DocumentFilter{SaleID: &saleID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.Documents, 1)
	return page.Documents[0]
}

// reload reads a document again from the store.
func (e *env) reload(t *testing.T, id uuid.UUID) *domain.Document {
	t.Helper()
	d, err := e.svc.GetDocument(t.Context(), e.actor, id)
	require.NoError(t, err)
	return d
}

// span returns the attributes of the last ended span named name.
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
// attribute key emits value ("" counts every point).
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
				if v, ok := dp.Attributes.Value(attribute.Key(key)); key == "" || ok && v.String() == value {
					total += dp.Value
				}
			}
		}
	}
	return total
}

// histogramCount returns how many values the float64 histogram recorded.
func (e *env) histogramCount(t *testing.T, name string) uint64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var total uint64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if h, ok := m.Data.(metricdata.Histogram[float64]); ok && m.Name == name {
				for _, dp := range h.DataPoints {
					total += dp.Count
				}
			}
		}
	}
	return total
}
