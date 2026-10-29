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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc       *app.Service
	store     *store
	customers *fakeCustomers
	drawer    *fakeDrawer
	actor     app.Actor
	now       time.Time
	spans     *tracetest.SpanRecorder
	metrics   *sdkmetric.ManualReader
}

// testNow is 2026-10-28 at 21:00 in Bogotá, already the 29th in UTC.
var testNow = time.Date(2026, 10, 29, 2, 0, 0, 0, time.UTC)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		store:     newStore(),
		customers: &fakeCustomers{names: map[uuid.UUID]string{}},
		actor:     app.Actor{TenantID: newID(), UserID: newID()},
		now:       testNow,
		spans:     tracetest.NewSpanRecorder(), metrics: sdkmetric.NewManualReader(),
	}
	e.drawer = &fakeDrawer{store: e.store, session: newID()}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Credit:         fakeCredit{e.store},
		Customers:      e.customers,
		Cash:           e.drawer,
		Now:            func() time.Time { return e.now },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// customer registers a customer of the actor's tenant in the directory.
func (e *env) customer(name string) uuid.UUID {
	id := newID()
	e.customers.names[id] = name
	return id
}

// creditCustomer registers a customer with a limit and a term.
func (e *env) creditCustomer(t *testing.T, name, limit string, days int) uuid.UUID {
	t.Helper()
	id := e.customer(name)
	_, err := e.svc.SetCreditTerms(t.Context(), e.actor, id, app.TermsRequest{Limit: dec(limit), Days: days})
	require.NoError(t, err)
	return id
}

// charge sells amount on credit to customerID, at e.now.
func (e *env) charge(t *testing.T, customerID uuid.UUID, amount string) *domain.Receivable {
	t.Helper()
	r, err := e.svc.Charge(t.Context(), e.actor, app.ChargeRequest{CustomerID: customerID, SaleID: newID(), Amount: dec(amount)})
	require.NoError(t, err)
	return r
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
