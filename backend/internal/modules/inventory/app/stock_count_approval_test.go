package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestApproveStockCount(t *testing.T) {
	t.Run("los ajustes solo se crean al aprobar y quedan en el kardex con referencia a la sesión", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.advance(time.Minute)
		e.countAs(t, e.actor, c.ID(), cable, "96.5")
		e.countAs(t, e.colleague(), c.ID(), shirt, "10")
		e.advance(time.Minute)
		e.sell(t, cable, location.ID(), "6.5")
		before := len(e.store.movements)

		approved, err := e.svc.ApproveStockCount(t.Context(), e.actor, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountApproved, approved.Status())
		require.Equal(t, e.actor.UserID, approved.ClosedBy())

		require.Len(t, e.store.movements, before+1, "solo la línea con diferencia")
		m := e.store.movements[before]
		require.Equal(t, domain.MovementAdjustment, m.Type())
		require.Equal(t, cable, m.VariantID())
		requireDecimal(t, "-3.5", m.Quantity())
		require.Equal(t, domain.CountAdjustmentReason, m.Reason())
		require.Equal(t, &domain.DocumentRef{Kind: domain.DocumentStockCount, ID: c.ID()}, m.Reference())
		// What was counted, minus what was sold after counting it.
		requireDecimal(t, "90", e.level(cable, location.ID()).Quantity)

		summary, ok := approved.Summary()
		require.True(t, ok)
		require.Equal(t, 2, summary.Counted)
		require.Equal(t, 1, summary.Exact)
	})

	t.Run("no aprueba con variantes en revisión y no ajusta nada", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountReview)
		e.countAs(t, e.actor, c.ID(), shirt, "7")
		e.countAs(t, e.actor, c.ID(), cable, "90")
		e.countAs(t, e.colleague(), c.ID(), cable, "100")
		before := len(e.store.movements)

		_, err := e.svc.ApproveStockCount(t.Context(), e.actor, c.ID())
		require.ErrorIs(t, err, domain.ErrCountNeedsReview)
		require.Len(t, e.store.movements, before)
		got, err := e.svc.GetStockCount(t.Context(), e.actor, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountOpen, got.Status())
	})

	t.Run("si un ajuste deja el stock negativo y la empresa no lo permite, no aprueba nada", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.advance(time.Minute)
		// The cable (created first) is adjusted first; the T-shirt then fails.
		e.countAs(t, e.actor, c.ID(), cable, "102")
		e.countAs(t, e.actor, c.ID(), shirt, "0")
		e.advance(time.Minute)
		e.sell(t, shirt, location.ID(), "9")
		before := len(e.store.movements)

		_, err := e.svc.ApproveStockCount(t.Context(), e.actor, c.ID())
		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		var stockErr *app.VariantStockError
		require.ErrorAs(t, err, &stockErr)
		require.Equal(t, shirt, stockErr.VariantID)
		require.Len(t, e.store.movements, before)
		requireDecimal(t, "100", e.level(cable, location.ID()).Quantity)
	})

	t.Run("cuenta la aprobación, sus ajustes y la exactitud", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.countAs(t, e.actor, c.ID(), cable, "99")
		e.countAs(t, e.actor, c.ID(), shirt, "10")
		_, err := e.svc.ApproveStockCount(t.Context(), e.actor, c.ID())
		require.NoError(t, err)

		_, attrs := e.span(t, "inventory.ApproveStockCount")
		require.Equal(t, c.ID().String(), attrs["stock_count.id"].AsString())
		require.Equal(t, int64(1), attrs["adjustments"].AsInt64())
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.stock_counts", "result", "approved"))
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.movements", "type", "adjustment"))
		accuracy := e.histogram(t, "cuadra.inventory.count_accuracy")
		require.Equal(t, uint64(1), accuracy.Count)
		require.InDelta(t, 0.5, accuracy.Sum, 1e-9)
	})
}

func TestCancelStockCount(t *testing.T) {
	t.Run("una sesión cancelada no genera ajustes", func(t *testing.T) {
		e, location, cable, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.countAs(t, e.actor, c.ID(), cable, "1")
		before := len(e.store.movements)

		canceled, err := e.svc.CancelStockCount(t.Context(), e.actor, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountCanceled, canceled.Status())
		require.Len(t, e.store.movements, before)

		_, err = e.svc.ApproveStockCount(t.Context(), e.actor, c.ID())
		require.ErrorIs(t, err, domain.ErrCountNotOpen)
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.stock_counts", "result", "canceled"))
		e.span(t, "inventory.CancelStockCount")
	})

	t.Run("después de cancelar se puede iniciar otro conteo en la sede", func(t *testing.T) {
		e, location, _, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		_, err := e.svc.CancelStockCount(t.Context(), e.actor, c.ID())
		require.NoError(t, err)
		e.startCount(t, location.ID(), domain.CountSum)
	})
}

// histogram returns the only data point of a float64 histogram.
func (e *env) histogram(t *testing.T, name string) metricdata.HistogramDataPoint[float64] {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if h, ok := m.Data.(metricdata.Histogram[float64]); ok && m.Name == name {
				require.Len(t, h.DataPoints, 1)
				return h.DataPoints[0]
			}
		}
	}
	require.FailNow(t, "histogram not recorded", name)
	return metricdata.HistogramDataPoint[float64]{}
}
