package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestSetReorderRules(t *testing.T) {
	t.Run("guarda o reemplaza las reglas de varias variantes en una sede", func(t *testing.T) {
		tests := []struct {
			business string
			product  int
		}{
			{"ferretería: tornillos por unidad", 0},
			{"almacén de ropa: un zapato en dos tallas", 1},
		}
		for _, tt := range tests {
			t.Run(tt.business, func(t *testing.T) {
				e := newEnv(t)
				loc := e.createLocation(t, "Bodega")
				_, variants := e.addProduct(tt.product)
				e.setRule(t, variants[0], loc.ID(), "1", "2")
				var lines []app.ReorderLine
				for _, v := range variants {
					lines = append(lines, app.ReorderLine{VariantID: v, Min: dec("2.5"), Max: dec("10")})
				}

				require.NoError(t, e.svc.SetReorderRules(t.Context(), e.actor, loc.ID(), lines))

				require.Len(t, e.store.rules, len(variants))
				for _, r := range e.store.rules {
					require.True(t, dec("2.5").Equal(r.Min()))
					require.True(t, dec("10").Equal(r.Max()))
				}
				e.span(t, "inventory.SetReorderRules")
				require.EqualValues(t, 1+len(variants), e.counter(t, "cuadra.inventory.reorder_rule_changes", "action", "set"))
			})
		}
	})

	t.Run("un rango inválido no guarda ninguna regla", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(1)
		lines := []app.ReorderLine{
			{VariantID: variants[0], Min: dec("1"), Max: dec("5")},
			{VariantID: variants[1], Min: dec("9"), Max: dec("5")},
		}

		err := e.svc.SetReorderRules(t.Context(), e.actor, loc.ID(), lines)
		require.ErrorIs(t, err, domain.ErrInvalidReorderRange)
		require.Empty(t, e.store.rules)
	})

	t.Run("rechaza una sede inexistente o inactiva", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(0)
		lines := []app.ReorderLine{{VariantID: variants[0], Min: dec("1"), Max: dec("5")}}

		require.ErrorIs(t, e.svc.SetReorderRules(t.Context(), e.actor, newID(), lines), domain.ErrLocationNotFound)
		e.deactivate(loc.ID())
		require.ErrorIs(t, e.svc.SetReorderRules(t.Context(), e.actor, loc.ID(), lines), domain.ErrLocationInactive)
		require.Empty(t, e.store.rules)
	})
}
