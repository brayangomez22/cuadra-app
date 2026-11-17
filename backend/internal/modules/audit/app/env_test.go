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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/app"
)

// env is a Service wired to a fake log, with recorders for spans and
// metrics.
type env struct {
	svc     *app.Service
	events  *fakeEvents
	actor   app.Actor
	now     time.Time
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

var testNow = time.Date(2026, 11, 15, 15, 0, 0, 0, time.UTC)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		events: &fakeEvents{},
		actor:  app.Actor{TenantID: newID(), UserID: newID(), Role: "admin"},
		now:    testNow,
		spans:  tracetest.NewSpanRecorder(), metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Events: e.events, Now: func() time.Time { return e.now },
		TracerProvider: tp, MeterProvider: mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// spanAttrs returns the attributes of the last ended span named name.
func (e *env) spanAttrs(t *testing.T, name string) map[attribute.Key]attribute.Value {
	t.Helper()
	var last sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			last = s
		}
	}
	require.NotNil(t, last, "span %s", name)
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range last.Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return attrs
}

// counter returns the sum of an int64 counter's points whose attribute key
// equals value.
func (e *env) counter(t *testing.T, name, key, value string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			sum, ok := m.Data.(metricdata.Sum[int64])
			if m.Name != name || !ok {
				continue
			}
			for _, dp := range sum.DataPoints {
				if v, ok := dp.Attributes.Value(attribute.Key(key)); ok && v.AsString() == value {
					total += dp.Value
				}
			}
		}
	}
	return total
}
