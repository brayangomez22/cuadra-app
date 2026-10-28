package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// sell takes qty units of a variant out of a location, as a sale.
func (e *env) sell(t *testing.T, variantID, locationID uuid.UUID, qty string) {
	t.Helper()
	_, err := e.svc.IssueSaleStock(t.Context(), e.actor, app.SaleStockInput{
		LocationID: locationID, SaleID: newID(),
		Lines: []app.SaleStockLine{{VariantID: variantID, Quantity: dec(qty)}},
	})
	require.NoError(t, err)
}

func TestStockCountLines(t *testing.T) {
	t.Run("la diferencia descuenta las ventas ocurridas durante el conteo", func(t *testing.T) {
		e, location, cable, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.advance(time.Minute)
		e.sell(t, cable, location.ID(), "12.5")
		e.advance(time.Minute)
		e.countAs(t, e.actor, c.ID(), cable, "85")

		line := e.lines(t, c.ID())[cable]
		require.Equal(t, domain.LineDifference, line.Status)
		requireDecimal(t, "87.5", line.Expected)
		requireDecimal(t, "-2.5", line.Difference)
		requireDecimal(t, "-6250", line.Value)
		require.Equal(t, e.store.variants[cable].productID, line.ProductID)
	})

	t.Run("una venta después del último conteo no cambia lo esperado", func(t *testing.T) {
		e, location, _, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.advance(time.Minute)
		e.countAs(t, e.actor, c.ID(), shirt, "10")
		e.advance(time.Minute)
		e.sell(t, shirt, location.ID(), "3")

		line := e.lines(t, c.ID())[shirt]
		require.Equal(t, domain.LineExact, line.Status)
		requireDecimal(t, "10", line.Expected)
	})

	t.Run("filtra las líneas por estado y resume el conteo", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.countAs(t, e.actor, c.ID(), cable, "98")
		e.countAs(t, e.actor, c.ID(), shirt, "10")

		differing := domain.LineDifference
		page, err := e.svc.StockCountLines(t.Context(), e.actor, c.ID(), app.LineFilter{Status: &differing, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, cable, page.Items[0].VariantID)
		require.Equal(t, 2, page.Summary.Lines)
		require.Equal(t, 2, page.Summary.Counted)
		require.Equal(t, 1, page.Summary.Exact)
		requireDecimal(t, "-5000", page.Summary.NetValue)

		_, attrs := e.span(t, "inventory.StockCountLines")
		require.Equal(t, c.ID().String(), attrs["stock_count.id"].AsString())
	})

	t.Run("pagina las líneas con las de mayor valor primero", func(t *testing.T) {
		e, location, cable, shirt := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		e.countAs(t, e.actor, c.ID(), cable, "99")
		e.countAs(t, e.actor, c.ID(), shirt, "9")

		page, err := e.svc.StockCountLines(t.Context(), e.actor, c.ID(), app.LineFilter{Limit: 1})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.Len(t, page.Items, 1)
		require.Equal(t, shirt, page.Items[0].VariantID, "18.000 pesos pesan más que 2.500")
	})
}

func TestRequestRecount(t *testing.T) {
	t.Run("un recuento descarta los conteos de rondas anteriores", func(t *testing.T) {
		e, location, cable, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountReview)
		e.countAs(t, e.actor, c.ID(), cable, "90")
		e.countAs(t, e.colleague(), c.ID(), cable, "100")
		require.Equal(t, domain.LineReview, e.lines(t, c.ID())[cable].Status)

		require.NoError(t, e.svc.RequestRecount(t.Context(), e.actor, c.ID(), []uuid.UUID{cable}))
		require.Equal(t, domain.LineUncounted, e.lines(t, c.ID())[cable].Status)

		e.countAs(t, e.actor, c.ID(), cable, "100")
		require.Equal(t, domain.LineExact, e.lines(t, c.ID())[cable].Status)
		e.span(t, "inventory.RequestRecount")
	})

	t.Run("rechaza recontar una variante que no está en el conteo", func(t *testing.T) {
		e, location, _, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		err := e.svc.RequestRecount(t.Context(), e.actor, c.ID(), []uuid.UUID{newID()})
		require.ErrorIs(t, err, domain.ErrVariantNotInCount)
	})

	t.Run("rechaza recontar en un conteo cerrado", func(t *testing.T) {
		e, location, cable, _ := stocked(t)
		c := e.startCount(t, location.ID(), domain.CountSum)
		_, err := e.svc.CancelStockCount(t.Context(), e.actor, c.ID())
		require.NoError(t, err)
		err = e.svc.RequestRecount(t.Context(), e.actor, c.ID(), []uuid.UUID{cable})
		require.ErrorIs(t, err, domain.ErrCountNotOpen)
	})
}
