package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestAuditOfStockAdjustments(t *testing.T) {
	adjustments := []struct {
		name, quantity string
	}{
		{"ferretería: se dañaron 2,5 metros de cable", "-2.5"},
		{"almacén de ropa: aparecen 3 camisas", "3"},
	}
	for _, tc := range adjustments {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			loc := e.createLocation(t, "Principal")
			_, variants := e.addProduct(0)
			e.receive(t, variants[0], loc.ID(), "10", "1000")

			_, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
				VariantID: variants[0], LocationID: loc.ID(), Quantity: dec(tc.quantity), Reason: "Revisión de bodega",
			})
			require.NoError(t, err)

			events := e.audit.of("stock.adjusted")
			require.Len(t, events, 1)
			require.Equal(t, variants[0], events[0].EntityID)
			require.Equal(t, loc.ID(), events[0].LocationID)
			require.Equal(t, map[string]string{"quantity": tc.quantity, "source": "adjustment"}, events[0].Details)
		})
	}

	t.Run("un ajuste rechazado no deja evento", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Principal")
		_, variants := e.addProduct(0)

		_, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			VariantID: variants[0], LocationID: loc.ID(), Quantity: dec("-1"), Reason: "Revisión",
		})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		require.Empty(t, e.audit.entries)
	})

	t.Run("si el registro falla, el ajuste falla", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Principal")
		_, variants := e.addProduct(0)
		e.audit.fail = errors.New("audit down")

		_, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			VariantID: variants[0], LocationID: loc.ID(), Quantity: dec("1"), Reason: "Revisión",
		})
		require.ErrorContains(t, err, "audit down")
	})

	t.Run("aprobar un conteo deja un evento por cada línea ajustada", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.countAs(t, e.actor, c.ID(), cable, "96.5")
		e.countAs(t, e.actor, c.ID(), shirt, "10")

		_, err := e.svc.ApproveStockCount(t.Context(), e.actor, c.ID())
		require.NoError(t, err)

		events := e.audit.of("stock.adjusted")
		require.Len(t, events, 1, "la camisa quedó exacta")
		require.Equal(t, cable, events[0].EntityID)
		require.Equal(t, location.ID(), events[0].LocationID)
		require.Equal(t, map[string]string{"quantity": "-3.5", "source": "stock_count"}, events[0].Details)
	})
}
