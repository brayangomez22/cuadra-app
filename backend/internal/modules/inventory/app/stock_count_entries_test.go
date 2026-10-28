package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestRecordCountEntry(t *testing.T) {
	t.Run("registra lo contado con el producto de la variante", func(t *testing.T) {
		e, location, cable, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)

		entry := e.countAs(t, e.actor, c.ID(), cable, "97.5")
		requireDecimal(t, "97.5", entry.Quantity())
		require.Equal(t, e.actor.UserID, entry.CountedBy())
		require.Equal(t, e.store.variants[cable].productID, entry.ProductID)

		_, attrs := e.span(t, "inventory.RecordCountEntry")
		require.Equal(t, c.ID().String(), attrs["stock_count.id"].AsString())
		require.Equal(t, cable.String(), attrs["variant.id"].AsString())
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.count_entries", "", ""))
	})

	t.Run("rechaza contar una variante fuera de un conteo parcial", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		clothes := newID()
		e.store.categories[clothes] = []uuid.UUID{shirt}
		c := e.startCount(t, location.ID(), domain.CountSum, clothes)

		_, err := e.svc.RecordCountEntry(t.Context(), e.actor, app.CountEntryInput{
			CountID: c.ID(), VariantID: cable, Quantity: dec("3"),
		})
		require.ErrorIs(t, err, domain.ErrVariantNotInCount)
	})

	t.Run("agrega la variante con inicio cero en un conteo total", func(t *testing.T) {
		e, location, _, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		_, notebooks := e.addProduct(0)

		e.countAs(t, e.actor, c.ID(), notebooks[0], "24")
		line := e.lines(t, c.ID())[notebooks[0]]
		requireDecimal(t, "0", line.Expected)
		requireDecimal(t, "24", line.Difference)
	})

	t.Run("convierte el conteo en unidad alternativa a la unidad base", func(t *testing.T) {
		e, location, _, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		dozen := newID()
		e.store.units[unitKey{shirt, dozen}] = dec("12")

		entry, err := e.svc.RecordCountEntry(t.Context(), e.actor, app.CountEntryInput{
			CountID: c.ID(), VariantID: shirt, Quantity: dec("1.5"), UnitID: &dozen,
		})
		require.NoError(t, err)
		requireDecimal(t, "18", entry.Quantity())
	})

	t.Run("rechaza una unidad que la variante no tiene", func(t *testing.T) {
		e, location, cable, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		_, err := e.svc.RecordCountEntry(t.Context(), e.actor, app.CountEntryInput{
			CountID: c.ID(), VariantID: cable, Quantity: dec("1"), UnitID: new(newID()),
		})
		require.ErrorIs(t, err, domain.ErrUnitNotAvailable)
	})

	t.Run("rechaza contar en un conteo de otra empresa", func(t *testing.T) {
		e, location, cable, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		_, err := e.svc.RecordCountEntry(t.Context(), stranger(), app.CountEntryInput{
			CountID: c.ID(), VariantID: cable, Quantity: dec("1"),
		})
		require.ErrorIs(t, err, domain.ErrCountNotFound)
	})
}

func TestCountEntriesOfEachCounter(t *testing.T) {
	e, location, cable, shirt := stocked(t)
	c := e.startCount(t, location.ID(), domain.CountSum)
	other := e.colleague()
	mine := e.countAs(t, e.actor, c.ID(), cable, "40")
	theirs := e.countAs(t, other, c.ID(), cable, "60")
	e.countAs(t, other, c.ID(), shirt, "10")

	t.Run("quien cuenta solo ve sus conteos", func(t *testing.T) {
		page, err := e.svc.ListCountEntries(t.Context(), other, domain.EntryFilter{CountID: c.ID(), Limit: 50}, false)
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		for _, entry := range page.Items {
			require.Equal(t, other.UserID, entry.CountedBy())
		}
	})

	t.Run("quien administra ve los conteos de todos, por variante", func(t *testing.T) {
		page, err := e.svc.ListCountEntries(t.Context(), e.actor,
			domain.EntryFilter{CountID: c.ID(), VariantID: &cable, Limit: 50}, true)
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
	})

	t.Run("solo borra el conteo propio salvo quien administra", func(t *testing.T) {
		err := e.svc.DeleteCountEntry(t.Context(), other, c.ID(), mine.ID(), false)
		require.ErrorIs(t, err, domain.ErrCountEntryNotYours)

		require.NoError(t, e.svc.DeleteCountEntry(t.Context(), other, c.ID(), theirs.ID(), false))
		require.NoError(t, e.svc.DeleteCountEntry(t.Context(), e.actor, c.ID(), mine.ID(), true))
		require.Equal(t, domain.LineUncounted, e.lines(t, c.ID())[cable].Status)
	})

	t.Run("un conteo que no existe no se borra", func(t *testing.T) {
		err := e.svc.DeleteCountEntry(t.Context(), e.actor, c.ID(), newID(), true)
		require.ErrorIs(t, err, domain.ErrCountEntryNotFound)
	})
}
