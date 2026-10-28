package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc     *app.Service
	store   *store
	actor   app.Actor
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

var testNow = time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		store: &store{
			customers: map[uuid.UUID]domain.CustomerSnapshot{},
			segments:  map[uuid.UUID]domain.SegmentSnapshot{},
		},
		actor: app.Actor{TenantID: newID(), UserID: newID()},
		spans: tracetest.NewSpanRecorder(), metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Customers:      fakeCustomers{e.store},
		Segments:       fakeSegments{e.store},
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// person is a customer identified by cédula, in no segment.
func person(document, name string) domain.CustomerParams {
	return domain.CustomerParams{
		PersonType: domain.PersonNatural, DocumentType: domain.DocumentCC, DocumentNumber: document,
		Name: name, Active: true,
	}
}

// company is a customer identified by NIT.
func company(nit, name string) domain.CustomerParams {
	return domain.CustomerParams{
		PersonType: domain.PersonLegal, DocumentType: domain.DocumentNIT, DocumentNumber: nit,
		Name: name, Active: true,
	}
}

// addSegment stores a segment of tenantID directly in the fake.
func (e *env) addSegment(t *testing.T, tenantID uuid.UUID, name string) *domain.Segment {
	t.Helper()
	s, err := domain.NewSegment(tenantID, name, testNow)
	require.NoError(t, err)
	require.NoError(t, fakeSegments{e.store}.Create(t.Context(), s))
	return s
}

// addCustomer creates a customer of the actor through the service.
func (e *env) addCustomer(t *testing.T, p domain.CustomerParams) *domain.Customer {
	t.Helper()
	c, err := e.svc.CreateCustomer(t.Context(), e.actor, p)
	require.NoError(t, err)
	return c
}

// span returns the attributes of the last ended span named name.
func (e *env) span(t *testing.T, name string) (sdktrace.ReadOnlySpan, map[attribute.Key]attribute.Value) {
	t.Helper()
	var found []sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			found = append(found, s)
		}
	}
	require.NotEmpty(t, found, "spans named %s", name)
	last := found[len(found)-1]
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range last.Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return last, attrs
}

// counter returns the sum of the data points of an int64 counter whose
// attribute key equals value.
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
				if v, ok := dp.Attributes.Value(attribute.Key(key)); ok && v.AsString() == value {
					total += dp.Value
				}
			}
		}
	}
	return total
}
