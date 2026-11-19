package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func TestIdleStock(t *testing.T) {
	t.Run("cuenta los días desde hoy en Colombia y ordena por plata quieta", func(t *testing.T) {
		e := newEnv(t)
		r, err := e.svc.IdleStock(t.Context(), e.actor, app.IdleQuery{Days: 30, Limit: 20})
		require.NoError(t, err)
		f := e.readers.idleFilter
		require.Equal(t, calendar.NewDate(2026, time.October, 20), f.Since.Day, "hoy es el 19 en Colombia")
		require.Equal(t, 30, r.Since.Days)
		require.True(t, r.Costs)
	})
	t.Run("sin ver costos ordena por cantidad y oculta el valor", func(t *testing.T) {
		e := newEnv(t)
		few, many := newID(), newID()
		e.readers.idle = []domain.IdleLine{
			{ProductID: few, ProductName: "Taladro", Quantity: dec("2"), Value: dec("900000")},
			{ProductID: many, ProductName: "Medias tobilleras", Quantity: dec("60"), Value: dec("180000")},
		}
		r, err := e.svc.IdleStock(t.Context(), e.cashier(), app.IdleQuery{Days: 30, Limit: 20})
		require.NoError(t, err)
		require.Equal(t, many, r.Rows[0].ProductID, "por cantidad")
		require.True(t, r.Value.IsZero())
		require.True(t, r.Rows[0].Value.IsZero())
	})
	t.Run("rechaza cero días", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.IdleStock(t.Context(), e.actor, app.IdleQuery{Days: 0, Limit: 20})
		require.ErrorIs(t, err, domain.ErrInvalidFilter)
	})
}

func TestStockAdjustments(t *testing.T) {
	rows := []domain.AdjustmentRow{
		{Key: "a", Label: "Tornillo 1/4", Adjustments: 2, Value: dec("-12000")},
		{Key: "b", Label: "Jean clásico (32)", Adjustments: 1, Value: dec("45000")},
	}
	t.Run("agrupa por variante y suma los totales", func(t *testing.T) {
		e := newEnv(t)
		e.readers.adjustments = domain.AdjustmentsReport{Rows: rows}
		r, err := e.svc.StockAdjustments(t.Context(), e.actor, app.AdjustmentsQuery{Range: nov(1, 19), Group: "variant"})
		require.NoError(t, err)
		require.Equal(t, domain.AdjustmentsByVariant, e.readers.adjustmentFilter.Group)
		require.Equal(t, 3, r.Adjustments)
		require.True(t, dec("33000").Equal(r.Value))
		require.True(t, r.Costs)
	})
	t.Run("oculta el valor a quien no ve costos", func(t *testing.T) {
		e := newEnv(t)
		e.readers.adjustments = domain.AdjustmentsReport{Rows: append([]domain.AdjustmentRow(nil), rows...)}
		r, err := e.svc.StockAdjustments(t.Context(), e.cashier(), app.AdjustmentsQuery{Range: nov(1, 19)})
		require.NoError(t, err)
		require.True(t, r.Value.IsZero())
		require.Equal(t, 3, r.Adjustments)
	})
}
