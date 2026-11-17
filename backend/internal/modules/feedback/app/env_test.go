package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
)

var now = time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)

type env struct {
	svc      *app.Service
	repo     *fakeRepository
	notifier *fakeNotifier
	spans    *tracetest.SpanRecorder
	metrics  *sdkmetric.ManualReader
	actor    app.Actor
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		repo:     newFakeRepository(),
		notifier: &fakeNotifier{},
		spans:    tracetest.NewSpanRecorder(),
		metrics:  sdkmetric.NewManualReader(),
		actor:    app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())},
	}
	svc, err := app.NewService(app.Deps{
		Feedback:       e.repo,
		Notifier:       e.notifier,
		Now:            func() time.Time { return now },
		TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans)),
		MeterProvider:  sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics)),
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

func (e *env) span(t *testing.T, name string) {
	t.Helper()
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			return
		}
	}
	t.Fatalf("no span %q", name)
}

// counted sums the data points of an Int64 counter.
func (e *env) counted(t *testing.T, name string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(context.Background(), &rm))
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
					total += dp.Value
				}
			}
		}
	}
	return total
}
