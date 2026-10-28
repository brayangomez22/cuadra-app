package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestReceiveInitialStock(t *testing.T) {
	t.Run("registra una entrada por variante, al costo de cada una", func(t *testing.T) {
		tests := []struct {
			business   string
			quantities []string
		}{
			{"ferretería: metros de cable y tubos", []string{"500.5", "120"}},
			{"almacén de ropa: camisetas por talla", []string{"10", "8"}},
		}
		for _, tt := range tests {
			t.Run(tt.business, func(t *testing.T) {
				e := newEnv(t)
				loc := e.createLocation(t, "Bodega")
				lines := []app.InitialStockLine{
					{VariantID: newID(), Quantity: dec(tt.quantities[0]), UnitCost: dec("1000")},
					{VariantID: newID(), Quantity: dec(tt.quantities[1]), UnitCost: dec("2500")},
				}

				require.NoError(t, e.svc.ReceiveInitialStock(t.Context(), e.actor, loc.ID(), lines))

				for _, l := range lines {
					level := e.level(l.VariantID, loc.ID())
					require.True(t, l.Quantity.Equal(level.Quantity))
					require.True(t, l.UnitCost.Equal(level.AverageCost))
				}
				require.EqualValues(t, 2, e.counter(t, "cuadra.inventory.movements", "type", string(domain.MovementInitialIn)))
				e.span(t, "inventory.ReceiveInitialStock")
			})
		}
	})

	t.Run("rechaza una sede inactiva sin registrar nada", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		e.deactivate(loc.ID())
		variant := newID()

		err := e.svc.ReceiveInitialStock(t.Context(), e.actor, loc.ID(),
			[]app.InitialStockLine{{VariantID: variant, Quantity: dec("1"), UnitCost: dec("1")}})
		require.ErrorIs(t, err, domain.ErrLocationInactive)
		require.True(t, e.level(variant, loc.ID()).Quantity.IsZero())
	})
}

func TestCheckLocation(t *testing.T) {
	t.Run("acepta una sede activa y rechaza una inexistente o de otra empresa", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")

		require.NoError(t, e.svc.CheckLocation(t.Context(), e.actor, loc.ID()))
		require.ErrorIs(t, e.svc.CheckLocation(t.Context(), e.actor, newID()), domain.ErrLocationNotFound)
		other := app.Actor{TenantID: newID(), UserID: newID()}
		require.ErrorIs(t, e.svc.CheckLocation(t.Context(), other, loc.ID()), domain.ErrLocationNotFound)
	})
}
