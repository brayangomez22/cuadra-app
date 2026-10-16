package httpx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func TestSetOperation(t *testing.T) {
	t.Run("marca la operación en el span y en la métrica de duración de otelhttp", func(t *testing.T) {
		spans := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		t.Cleanup(func() {
			_ = tp.Shutdown(context.Background())
			_ = mp.Shutdown(context.Background())
		})
		handler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpx.SetOperation(r.Context(), "pos.search")
			w.WriteHeader(http.StatusOK)
		}), "http.server", otelhttp.WithTracerProvider(tp), otelhttp.WithMeterProvider(mp))

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/products?q=x", nil))

		require.Len(t, spans.Ended(), 1)
		require.Contains(t, spans.Ended()[0].Attributes(), attribute.String("cuadra.operation", "pos.search"))

		var rm metricdata.ResourceMetrics
		require.NoError(t, reader.Collect(t.Context(), &rm))
		var found bool
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != "http.server.request.duration" {
					continue
				}
				hist, ok := m.Data.(metricdata.Histogram[float64])
				require.True(t, ok)
				for _, dp := range hist.DataPoints {
					v, ok := dp.Attributes.Value("cuadra.operation")
					found = found || ok && v.AsString() == "pos.search"
				}
			}
		}
		require.True(t, found, "http.server.request.duration has cuadra.operation=pos.search")
	})

	t.Run("sin otelhttp no hace nada", func(t *testing.T) {
		require.NotPanics(t, func() { httpx.SetOperation(t.Context(), "pos.search") })
	})
}
