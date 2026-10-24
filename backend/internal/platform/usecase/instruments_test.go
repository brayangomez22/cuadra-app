package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// failingMeter fails to create counters.
type failingMeter struct{ metricnoop.Meter }

var errMeter = errors.New("meter failed")

func (failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errMeter
}

func collect(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	out := map[string]metricdata.Metrics{}
	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			out[m.Name] = m
		}
	}
	return out
}

func TestCreate(t *testing.T) {
	t.Run("crea los contadores e histogramas con su nombre, descripción y unidad", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meter := usecase.Meter(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), "test/instrumentation")
		var sales metric.Int64Counter
		var amount metric.Float64Histogram

		err := usecase.Create(meter,
			usecase.Counter(&sales, "cuadra.sales.confirmed", "Sales confirmed.", "{sale}"),
			usecase.Histogram(&amount, "cuadra.sales.amount", "Total of the sales.", "{COP}", 1_000, 10_000),
		)
		require.NoError(t, err)
		sales.Add(context.Background(), 2)
		amount.Record(context.Background(), 5_000)

		metrics := collect(t, reader)
		require.Equal(t, "Sales confirmed.", metrics["cuadra.sales.confirmed"].Description)
		require.Equal(t, "{sale}", metrics["cuadra.sales.confirmed"].Unit)
		sum, ok := metrics["cuadra.sales.confirmed"].Data.(metricdata.Sum[int64])
		require.True(t, ok)
		require.Equal(t, int64(2), sum.DataPoints[0].Value)
		require.Equal(t, "{COP}", metrics["cuadra.sales.amount"].Unit)
		histogram, ok := metrics["cuadra.sales.amount"].Data.(metricdata.Histogram[float64])
		require.True(t, ok)
		require.Equal(t, []float64{1_000, 10_000}, histogram.DataPoints[0].Bounds)
	})

	t.Run("devuelve el error del primer instrumento que no se puede crear", func(t *testing.T) {
		var a, b metric.Int64Counter

		err := usecase.Create(failingMeter{},
			usecase.Counter(&a, "cuadra.x.a", "A.", "{a}"),
			usecase.Counter(&b, "cuadra.x.b", "B.", "{b}"),
		)

		require.ErrorIs(t, err, errMeter)
		require.Nil(t, a)
	})

	t.Run("sin proveedor de métricas no falla (no-op)", func(t *testing.T) {
		var c metric.Int64Counter

		err := usecase.Create(usecase.Meter(nil, "test/instrumentation"), usecase.Counter(&c, "cuadra.x.c", "C.", "{c}"))

		require.NoError(t, err)
		require.NotNil(t, c)
	})
}

func TestLogger(t *testing.T) {
	t.Run("sin logger devuelve uno que descarta", func(t *testing.T) {
		logger := usecase.Logger(nil)

		require.NotNil(t, logger)
		require.False(t, logger.Enabled(context.Background(), 0))
	})
}
