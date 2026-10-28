//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestCountEntryRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("en un conteo total agrega la línea con inicio cero", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		notebooks := f.createVariant(t, l.tenant)
		f.count(t, c, notebooks, newID(), "24", now.Add(time.Minute))

		line := f.sheet(t, c)[notebooks]
		require.True(t, line.StartQuantity.IsZero())
		require.True(t, dec("24").Equal(line.Counts[0].Quantity))
	})

	t.Run("en un conteo parcial rechaza una variante sin línea", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now, l.shirt)
		e, err := c.NewEntry(newID(), l.cable, dec("3"), newID(), now)
		require.NoError(t, err)
		require.ErrorIs(t, f.entries.Add(t.Context(), e, false), domain.ErrVariantNotInCount)
	})

	t.Run("rechaza una variante que no existe", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		e, err := c.NewEntry(newID(), newID(), dec("3"), newID(), now)
		require.NoError(t, err)
		require.ErrorIs(t, f.entries.Add(t.Context(), e, true), domain.ErrVariantNotFound)
	})

	t.Run("lista lo contado de la ronda vigente, por persona y variante, lo más reciente primero", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		ana, beto := newID(), newID()
		old := f.count(t, c, l.shirt, ana, "8", now.Add(time.Minute))
		require.NoError(t, f.counts.Recount(t.Context(), l.tenant, c.ID(), []uuid.UUID{l.shirt}))
		first := f.count(t, c, l.cable, ana, "40", now.Add(2*time.Minute))
		latest := f.count(t, c, l.shirt, ana, "10", now.Add(3*time.Minute))
		f.count(t, c, l.cable, beto, "60", now.Add(4*time.Minute))

		page, err := f.entries.List(t.Context(), l.tenant, domain.EntryFilter{CountID: c.ID(), CountedBy: &ana, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.Equal(t, []uuid.UUID{latest.ID(), first.ID()}, []uuid.UUID{page.Items[0].ID(), page.Items[1].ID()})
		require.NotContains(t, []uuid.UUID{page.Items[0].ID(), page.Items[1].ID()}, old.ID())

		byVariant, err := f.entries.List(t.Context(), l.tenant, domain.EntryFilter{CountID: c.ID(), VariantID: &l.cable, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 2, byVariant.Total)
	})

	t.Run("lee y borra lo contado", func(t *testing.T) {
		l := f.countedLocation(t)
		c, _ := f.startCount(t, l.tenant, l.location, now)
		e := f.count(t, c, l.cable, newID(), "12.75", now.Add(time.Minute))

		got, err := f.entries.Get(t.Context(), l.tenant, c.ID(), e.ID())
		require.NoError(t, err)
		require.True(t, dec("12.75").Equal(got.Quantity()))
		require.Equal(t, now.Add(time.Minute), got.CountedAt())

		require.NoError(t, f.entries.Delete(t.Context(), got))
		_, err = f.entries.Get(t.Context(), l.tenant, c.ID(), e.ID())
		require.ErrorIs(t, err, domain.ErrCountEntryNotFound)
		require.ErrorIs(t, f.entries.Delete(t.Context(), got), domain.ErrCountEntryNotFound)
	})
}

func TestStockCountList(t *testing.T) {
	f := newFixture(t)
	l := f.countedLocation(t)
	other := f.createLocation(t, l.tenant, "Bodega").ID()
	older, _ := f.startCount(t, l.tenant, l.location, now)
	require.NoError(t, f.counts.Close(t.Context(), l.tenant, older.ID(), func(_ context.Context, c *domain.StockCount) error {
		return c.Cancel(newID(), now)
	}))
	newer, _ := f.startCount(t, l.tenant, l.location, now.Add(time.Hour))
	f.startCount(t, l.tenant, other, now.Add(2*time.Hour))

	t.Run("lista los conteos de una sede, el más reciente primero", func(t *testing.T) {
		page, err := f.counts.List(t.Context(), l.tenant, domain.CountFilter{LocationID: &l.location, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.Equal(t, newer.ID(), page.Items[0].ID())
		require.Equal(t, older.ID(), page.Items[1].ID())
	})

	t.Run("filtra por estado", func(t *testing.T) {
		canceled := domain.CountCanceled
		page, err := f.counts.List(t.Context(), l.tenant, domain.CountFilter{Status: &canceled, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, domain.CountCanceled, page.Items[0].Status())
	})
}

func TestStockCountIsolation(t *testing.T) {
	f := newFixture(t)

	t.Run("aislamiento: el tenant B no ve ni modifica los conteos del A", func(t *testing.T) {
		a := f.countedLocation(t)
		b := f.createTenant(t)
		c, _ := f.startCount(t, a.tenant, a.location, now)
		e := f.count(t, c, a.cable, newID(), "5", now)

		_, err := f.counts.Get(t.Context(), b, c.ID())
		require.ErrorIs(t, err, domain.ErrCountNotFound)
		page, err := f.counts.List(t.Context(), b, domain.CountFilter{Limit: 10})
		require.NoError(t, err)
		require.Zero(t, page.Total)
		progress, err := f.counts.Progress(t.Context(), b, c.ID())
		require.NoError(t, err)
		require.Zero(t, progress.Lines)
		require.ErrorIs(t, f.counts.Recount(t.Context(), b, c.ID(), []uuid.UUID{a.cable}), domain.ErrVariantNotInCount)
		err = f.counts.Close(t.Context(), b, c.ID(), func(_ context.Context, c *domain.StockCount) error { return c.Cancel(newID(), now) })
		require.ErrorIs(t, err, domain.ErrCountNotFound)

		_, err = f.entries.Get(t.Context(), b, c.ID(), e.ID())
		require.ErrorIs(t, err, domain.ErrCountEntryNotFound)
		entries, err := f.entries.List(t.Context(), b, domain.EntryFilter{CountID: c.ID(), Limit: 10})
		require.NoError(t, err)
		require.Zero(t, entries.Total)

		// An entry of B pointing at A's count finds no count.
		forged, err := domain.RehydrateCountEntry(domain.CountEntrySnapshot{
			ID: newID(), TenantID: b, CountID: c.ID(), VariantID: a.cable, Quantity: dec("1"), CountedBy: newID(), CountedAt: now,
		})
		require.NoError(t, err)
		require.ErrorIs(t, f.entries.Add(t.Context(), forged, true), domain.ErrCountNotFound)

		got, err := f.counts.Get(t.Context(), a.tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountOpen, got.Status())
		require.Len(t, f.sheet(t, got)[a.cable].Counts, 1)
	})
}
