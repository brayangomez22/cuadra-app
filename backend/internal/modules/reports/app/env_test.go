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

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
)

// env is a Service wired to fake readers, with recorders for spans and
// metrics. Its actor is an owner who sees costs.
type env struct {
	svc     *app.Service
	readers *fakeReaders
	aging   *fakeAging
	actor   app.Actor
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

// testNow is 2026-11-19 at 9 p. m. in Colombia (the 20th in UTC).
var testNow = time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		readers: &fakeReaders{}, aging: &fakeAging{},
		actor: app.Actor{TenantID: newID(), UserID: newID(), Role: "owner", SeesCosts: true},
		spans: tracetest.NewSpanRecorder(), metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})
	svc, err := app.NewService(app.Deps{
		Sales: e.readers, Stock: e.readers, Team: e.readers, Receivables: e.aging,
		Now: func() time.Time { return testNow }, TracerProvider: tp, MeterProvider: mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

// cashier is an actor of the env's tenant who does not see costs.
func (e *env) cashier() app.Actor {
	return app.Actor{TenantID: e.actor.TenantID, UserID: newID(), Role: "admin", SeesCosts: false}
}

// nov is a range of days of November 2026.
func nov(from, to int) app.Range {
	return app.Range{From: calendar.NewDate(2026, time.November, from), To: calendar.NewDate(2026, time.November, to)}
}

// span returns the last ended span named name, or fails.
func (e *env) span(t *testing.T, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	var last sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			last = s
		}
	}
	require.NotNil(t, last, "span %s", name)
	return last
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
