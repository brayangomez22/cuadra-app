package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func reorderInput(variantID, locationID uuid.UUID, minQty, maxQty string) app.ReorderInput {
	return app.ReorderInput{VariantID: variantID, LocationID: locationID, Min: dec(minQty), Max: dec(maxQty)}
}

// setRule defines the minimum and maximum of a variant at a location.
func (e *env) setRule(t *testing.T, variantID, locationID uuid.UUID, minQty, maxQty string) {
	t.Helper()
	_, err := e.svc.SetReorderRule(t.Context(), e.actor, reorderInput(variantID, locationID, minQty, maxQty))
	require.NoError(t, err)
}

func TestSetReorderRule(t *testing.T) {
	t.Run("guarda el mínimo y el máximo de una variante en una sede", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(0)

		rule, err := e.svc.SetReorderRule(t.Context(), e.actor, reorderInput(variants[0], loc.ID(), "10", "50"))
		require.NoError(t, err)
		require.Equal(t, e.actor.TenantID, rule.TenantID())
		require.True(t, dec("10").Equal(rule.Min()))
		require.True(t, dec("50").Equal(rule.Max()))

		_, attrs := e.span(t, "inventory.SetReorderRule")
		require.Equal(t, variants[0].String(), attrs["variant.id"].AsString())
		require.Equal(t, loc.ID().String(), attrs["location.id"].AsString())
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.reorder_rule_changes", "action", "set"))
	})

	t.Run("reemplaza la regla que ya tenía", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Almacén")
		_, variants := e.addProduct(1)
		e.setRule(t, variants[1], loc.ID(), "2", "6")
		e.setRule(t, variants[1], loc.ID(), "3", "8")

		page, err := e.svc.ListStock(t.Context(), e.actor, app.StockQuery{LocationID: loc.ID()})
		require.NoError(t, err)
		require.Empty(t, page.Items)
		require.Len(t, e.store.rules, 1)
	})

	t.Run("rechaza un rango inválido sin guardar nada", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(0)

		_, err := e.svc.SetReorderRule(t.Context(), e.actor, reorderInput(variants[0], loc.ID(), "8", "5"))
		require.ErrorIs(t, err, domain.ErrInvalidReorderRange)
		require.Empty(t, e.store.rules)
	})

	t.Run("rechaza una sede o una variante que no existen en la empresa", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(0)

		_, err := e.svc.SetReorderRule(t.Context(), e.actor, reorderInput(variants[0], newID(), "1", "2"))
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
		_, err = e.svc.SetReorderRule(t.Context(), e.actor, reorderInput(newID(), loc.ID(), "1", "2"))
		require.ErrorIs(t, err, domain.ErrVariantNotFound)
	})
}

func TestDeleteReorderRule(t *testing.T) {
	t.Run("elimina la regla", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(0)
		e.setRule(t, variants[0], loc.ID(), "1", "4")

		require.NoError(t, e.svc.DeleteReorderRule(t.Context(), e.actor, variants[0], loc.ID()))
		require.Empty(t, e.store.rules)
		e.span(t, "inventory.DeleteReorderRule")
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.reorder_rule_changes", "action", "delete"))
	})

	t.Run("eliminar una regla que no existe falla con ErrReorderRuleNotFound", func(t *testing.T) {
		e := newEnv(t)
		err := e.svc.DeleteReorderRule(t.Context(), e.actor, newID(), newID())
		require.ErrorIs(t, err, domain.ErrReorderRuleNotFound)
	})
}

func TestListReorderAlerts(t *testing.T) {
	t.Run("lista solo las variantes con stock en el mínimo o por debajo, con lo que hay que pedir", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		hardware, screws := e.addProduct(0) // tornillos por unidad
		shoes, sizes := e.addProduct(1)     // un zapato en dos tallas
		e.receive(t, screws[0], loc.ID(), "10", "150")
		e.receive(t, sizes[0], loc.ID(), "1", "90000")
		e.receive(t, sizes[1], loc.ID(), "5", "90000")
		e.setRule(t, screws[0], loc.ID(), "10", "50")
		e.setRule(t, sizes[0], loc.ID(), "2", "6")
		e.setRule(t, sizes[1], loc.ID(), "2", "6")

		page, err := e.svc.ListReorderAlerts(t.Context(), e.actor, app.ReorderQuery{LocationID: new(loc.ID())})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.Equal(t, sizes[0], page.Items[0].Rule.VariantID())
		require.Equal(t, shoes, page.Items[0].ProductID)
		require.True(t, dec("5").Equal(page.Items[0].Suggested))
		require.Equal(t, screws[0], page.Items[1].Rule.VariantID())
		require.Equal(t, hardware, page.Items[1].ProductID)
		require.True(t, dec("40").Equal(page.Items[1].Suggested))
		e.span(t, "inventory.ListReorderAlerts")
	})

	t.Run("sin sede lista las alertas de todas las sedes", func(t *testing.T) {
		e := newEnv(t)
		north, south := e.createLocation(t, "Norte"), e.createLocation(t, "Sur")
		_, variants := e.addProduct(0)
		e.setRule(t, variants[0], north.ID(), "1", "3")
		e.setRule(t, variants[0], south.ID(), "1", "3")

		page, err := e.svc.ListReorderAlerts(t.Context(), e.actor, app.ReorderQuery{Limit: 500, Offset: -1})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		got := e.store.reorderFilters[0]
		require.Nil(t, got.LocationID)
		require.Equal(t, 100, got.Limit)
		require.Equal(t, 0, got.Offset)
	})

	t.Run("las alertas de una sede inexistente fallan con ErrLocationNotFound", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ListReorderAlerts(t.Context(), e.actor, app.ReorderQuery{LocationID: new(newID())})
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})
}

func TestListStockWithReorder(t *testing.T) {
	t.Run("las existencias traen su regla y si hay que reponer", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		_, variants := e.addProduct(2)
		e.receive(t, variants[0], loc.ID(), "2.5", "1000")
		e.receive(t, variants[1], loc.ID(), "40", "1000")
		e.receive(t, variants[2], loc.ID(), "1", "1000")
		e.setRule(t, variants[0], loc.ID(), "3", "30")
		e.setRule(t, variants[1], loc.ID(), "3", "30")

		page, err := e.svc.ListStock(t.Context(), e.actor, app.StockQuery{LocationID: loc.ID()})
		require.NoError(t, err)
		byVariant := map[uuid.UUID]app.Level{}
		for _, l := range page.Items {
			byVariant[l.VariantID()] = l
		}
		require.NotNil(t, byVariant[variants[0]].Reorder)
		require.True(t, byVariant[variants[0]].NeedsReorder())
		require.NotNil(t, byVariant[variants[1]].Reorder)
		require.False(t, byVariant[variants[1]].NeedsReorder())
		require.Nil(t, byVariant[variants[2]].Reorder)
		require.False(t, byVariant[variants[2]].NeedsReorder())
	})
}
