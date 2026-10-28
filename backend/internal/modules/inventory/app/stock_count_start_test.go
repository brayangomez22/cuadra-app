package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// advance moves the service's clock forward.
func (e *env) advance(d time.Duration) { e.now = e.now.Add(d) }

// startCount starts a count at locationID: full without categories.
func (e *env) startCount(t *testing.T, locationID uuid.UUID, mode domain.CountMode, categories ...uuid.UUID) app.CountView {
	t.Helper()
	c, err := e.svc.StartStockCount(t.Context(), e.actor, app.StartCountInput{
		LocationID: locationID, CategoryIDs: categories, Mode: mode,
	})
	require.NoError(t, err)
	return c
}

// countAs records that a counts qty of variantID in the count.
func (e *env) countAs(t *testing.T, a app.Actor, countID, variantID uuid.UUID, qty string) app.EntryView {
	t.Helper()
	entry, err := e.svc.RecordCountEntry(t.Context(), a, app.CountEntryInput{
		CountID: countID, VariantID: variantID, Quantity: dec(qty),
	})
	require.NoError(t, err)
	return entry
}

// colleague is another person of the actor's tenant.
func (e *env) colleague() app.Actor { return app.Actor{TenantID: e.actor.TenantID, UserID: newID()} }

// lines returns the evaluated lines of a count by variant.
func (e *env) lines(t *testing.T, countID uuid.UUID) map[uuid.UUID]app.LineView {
	t.Helper()
	page, err := e.svc.StockCountLines(t.Context(), e.actor, countID, app.LineFilter{Limit: 100})
	require.NoError(t, err)
	out := map[uuid.UUID]app.LineView{}
	for _, l := range page.Items {
		out[l.VariantID] = l
	}
	return out
}

// stocked is a location with a cable sold by meters and a T-shirt in size
// M, both with stock.
func stocked(t *testing.T) (e *env, location *domain.Location, cable, shirt uuid.UUID) {
	t.Helper()
	e = newEnv(t)
	location = e.createLocation(t, "Principal")
	_, cables := e.addProduct(0)
	_, shirts := e.addProduct(0)
	e.receive(t, cables[0], location.ID(), "100", "2500")
	e.receive(t, shirts[0], location.ID(), "10", "18000")
	return e, location, cables[0], shirts[0]
}

func TestStartStockCount(t *testing.T) {
	t.Run("inicia un conteo total con la foto de los niveles de la sede", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		other := e.createLocation(t, "Bodega")
		_, elsewhere := e.addProduct(0)
		e.receive(t, elsewhere[0], other.ID(), "5", "100")

		c := e.startCount(t, location.ID(), domain.CountSum)
		require.Equal(t, domain.CountOpen, c.Status())
		require.False(t, c.IsPartial())
		require.Equal(t, domain.CountProgress{Lines: 2}, c.Progress)

		lines := e.lines(t, c.ID())
		require.Len(t, lines, 2)
		requireDecimal(t, "100", lines[cable].Expected)
		requireDecimal(t, "10", lines[shirt].Expected)
		require.Equal(t, domain.LineUncounted, lines[cable].Status)
	})

	t.Run("inicia un conteo parcial con las variantes de las categorías", func(t *testing.T) {
		e, location, cable, _ := stocked(t)
		_, gloves := e.addProduct(0)
		tools := newID()
		e.store.categories[tools] = []uuid.UUID{cable, gloves[0]}

		c := e.startCount(t, location.ID(), domain.CountReview, tools)
		require.True(t, c.IsPartial())
		require.Equal(t, 2, c.Progress.Lines)
		lines := e.lines(t, c.ID())
		requireDecimal(t, "100", lines[cable].Expected)
		requireDecimal(t, "0", lines[gloves[0]].Expected)
	})

	t.Run("rechaza una categoría que no existe", func(t *testing.T) {
		e, location, _, _ := stocked(t)
		_, err := e.svc.StartStockCount(t.Context(), e.actor, app.StartCountInput{
			LocationID: location.ID(), CategoryIDs: []uuid.UUID{newID()}, Mode: domain.CountSum,
		})
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rechaza una segunda sesión abierta en la misma sede", func(t *testing.T) {
		e, location, _, _ := stocked(t)
		e.startCount(t, location.ID(), domain.CountSum)
		_, err := e.svc.StartStockCount(t.Context(), e.actor, app.StartCountInput{
			LocationID: location.ID(), Mode: domain.CountSum,
		})
		require.ErrorIs(t, err, domain.ErrCountInProgress)
	})

	t.Run("rechaza iniciar en una sede inactiva", func(t *testing.T) {
		e, location, _, _ := stocked(t)
		e.deactivate(location.ID())
		_, err := e.svc.StartStockCount(t.Context(), e.actor, app.StartCountInput{
			LocationID: location.ID(), Mode: domain.CountSum,
		})
		require.ErrorIs(t, err, domain.ErrLocationInactive)
	})

	t.Run("abre su span y cuenta la sesión iniciada", func(t *testing.T) {
		e, location, _, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)

		_, attrs := e.span(t, "inventory.StartStockCount")
		require.Equal(t, c.ID().String(), attrs["stock_count.id"].AsString())
		require.Equal(t, location.ID().String(), attrs["location.id"].AsString())
		require.Equal(t, int64(2), attrs["lines"].AsInt64())
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.stock_counts", "result", "started"))
	})
}

func TestGetAndListStockCounts(t *testing.T) {
	e, location, cable, _ := stocked(t)
	c := e.startCount(t, location.ID(), domain.CountSum)
	e.countAs(t, e.actor, c.ID(), cable, "99")

	t.Run("muestra el avance del conteo", func(t *testing.T) {
		got, err := e.svc.GetStockCount(t.Context(), e.actor, c.ID())
		require.NoError(t, err)
		require.Equal(t, domain.CountProgress{Lines: 2, Counted: 1}, got.Progress)
	})

	t.Run("un conteo de otra empresa no existe", func(t *testing.T) {
		_, err := e.svc.GetStockCount(t.Context(), stranger(), c.ID())
		require.ErrorIs(t, err, domain.ErrCountNotFound)
	})

	t.Run("lista los conteos abiertos de la sede", func(t *testing.T) {
		open := domain.CountOpen
		page, err := e.svc.ListStockCounts(t.Context(), e.actor, domain.CountFilter{
			LocationID: new(location.ID()), Status: &open, Limit: 20,
		})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, c.ID(), page.Items[0].ID())
	})
}

// stranger is someone of another tenant.
func stranger() app.Actor { return app.Actor{TenantID: newID(), UserID: newID()} }
