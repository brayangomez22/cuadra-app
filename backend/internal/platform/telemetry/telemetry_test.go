package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSetup(t *testing.T) {
	t.Run("con OTEL_SDK_DISABLED=true no instala proveedores y el shutdown no falla", func(t *testing.T) {
		getenv := func(key string) string {
			if key == "OTEL_SDK_DISABLED" {
				return "true"
			}
			return ""
		}

		shutdown, err := Setup(context.Background(), Resource{ServiceName: "cuadra-api"}, getenv)

		require.NoError(t, err)
		require.NotNil(t, shutdown)
		_, isSDK := otel.GetTracerProvider().(*sdktrace.TracerProvider)
		require.False(t, isSDK, "no debe instalar el TracerProvider del SDK")
		require.NoError(t, shutdown(context.Background()))
	})
}

func TestNewResource(t *testing.T) {
	t.Run("el recurso incluye nombre de servicio, versión y ambiente", func(t *testing.T) {
		res, err := newResource(context.Background(), Resource{
			ServiceName:    "cuadra-api",
			ServiceVersion: "abc123",
			Environment:    "development",
		})

		require.NoError(t, err)
		attrs := map[attribute.Key]string{}
		for _, kv := range res.Attributes() {
			attrs[kv.Key] = kv.Value.String()
		}
		require.Equal(t, "cuadra-api", attrs["service.name"])
		require.Equal(t, "abc123", attrs["service.version"])
		require.Equal(t, "development", attrs["deployment.environment.name"])
	})
}

func TestViews(t *testing.T) {
	// bounds records one value on a histogram named name, created as the
	// libraries do, and returns the bucket boundaries it was exported with.
	bounds := func(t *testing.T, name string, opts ...metric.Float64HistogramOption) []float64 {
		t.Helper()
		reader := sdkmetric.NewManualReader()
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithView(views()...))
		h, err := mp.Meter("test").Float64Histogram(name, opts...)
		require.NoError(t, err)
		h.Record(t.Context(), 0.2)

		var rm metricdata.ResourceMetrics
		require.NoError(t, reader.Collect(t.Context(), &rm))
		require.Len(t, rm.ScopeMetrics, 1)
		require.Len(t, rm.ScopeMetrics[0].Metrics, 1)
		data, ok := rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[float64])
		require.True(t, ok)
		require.Len(t, data.DataPoints, 1)
		return data.DataPoints[0].Bounds
	}

	t.Run("la duración de las requests HTTP tiene límites en 300 ms (SLO del POS) y 500 ms (alerta de p95)", func(t *testing.T) {
		// otelhttp sets its own boundaries, which skip 0.3.
		got := bounds(t, "http.server.request.duration",
			metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10))

		require.Contains(t, got, 0.3)
		require.Contains(t, got, 0.5)
		require.IsIncreasing(t, got)
	})

	t.Run("los demás histogramas conservan sus límites", func(t *testing.T) {
		got := bounds(t, "cuadra.sales.amount", metric.WithExplicitBucketBoundaries(10000, 50000))

		require.Equal(t, []float64{10000, 50000}, got)
	})
}
