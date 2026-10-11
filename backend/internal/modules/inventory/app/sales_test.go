package app_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// requireDecimal compares decimals by value, so 7 equals 7.0000.
func requireDecimal(t *testing.T, want string, got decimal.Decimal) {
	t.Helper()
	require.Truef(t, dec(want).Equal(got), "want %s, got %s", want, got)
}

func saleLine(productID uuid.UUID, qty string) app.SaleStockLine {
	return app.SaleStockLine{ProductID: productID, Quantity: dec(qty)}
}

func TestIssueSaleStock(t *testing.T) {
	t.Run("registra una salida por producto ordenadas por producto", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Sede")
		p1, p2 := newID(), newID()
		e.receive(t, p1, loc.ID(), "10", "1000")
		e.receive(t, p2, loc.ID(), "5", "200")
		saleID := newID()

		// p2 primero en la venta: las salidas igual van en orden de producto.
		movements, err := e.svc.IssueSaleStock(t.Context(), e.actor, app.SaleStockInput{
			LocationID: loc.ID(), SaleID: saleID,
			Lines: []app.SaleStockLine{saleLine(p2, "1.5"), saleLine(p1, "3")},
		})
		require.NoError(t, err)

		require.Len(t, movements, 2)
		first, second := movements[0].ProductID(), movements[1].ProductID()
		require.Negative(t, bytes.Compare(first[:], second[:]))
		for _, m := range movements {
			require.Equal(t, domain.MovementSaleOut, m.Type())
			require.Equal(t, &domain.DocumentRef{Kind: domain.DocumentSale, ID: saleID}, m.Reference())
			require.Equal(t, e.actor.UserID, m.UserID())
		}
		requireDecimal(t, "7", e.level(p1, loc.ID()).Quantity)
		requireDecimal(t, "3.5", e.level(p2, loc.ID()).Quantity)

		require.Equal(t, int64(2), e.counter(t, "cuadra.inventory.movements", "type", "sale_out"))
		_, attrs := e.span(t, "inventory.IssueSaleStock")
		require.Equal(t, saleID.String(), attrs["sale.id"].AsString())
		require.Equal(t, loc.ID().String(), attrs["location.id"].AsString())
	})

	t.Run("rechaza la salida en una sede inactiva", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Sede")
		p := newID()
		e.receive(t, p, loc.ID(), "10", "1000")
		e.deactivate(loc.ID())

		_, err := e.svc.IssueSaleStock(t.Context(), e.actor, app.SaleStockInput{
			LocationID: loc.ID(), SaleID: newID(), Lines: []app.SaleStockLine{saleLine(p, "1")},
		})
		require.ErrorIs(t, err, domain.ErrLocationInactive)
		requireDecimal(t, "10", e.level(p, loc.ID()).Quantity)
	})

	t.Run("rechaza una sede de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Sede")
		other := e.actor
		other.TenantID = newID()

		_, err := e.svc.IssueSaleStock(t.Context(), other, app.SaleStockInput{
			LocationID: loc.ID(), SaleID: newID(), Lines: []app.SaleStockLine{saleLine(newID(), "1")},
		})
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("si falta stock de un producto, el error dice cuál", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Sede")
		p := newID()
		e.receive(t, p, loc.ID(), "2", "1000")

		_, err := e.svc.IssueSaleStock(t.Context(), e.actor, app.SaleStockInput{
			LocationID: loc.ID(), SaleID: newID(), Lines: []app.SaleStockLine{saleLine(p, "2.5")},
		})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		var pe *app.ProductStockError
		require.True(t, errors.As(err, &pe))
		require.Equal(t, p, pe.ProductID)
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.insufficient_stock", "type", "sale_out"))
	})

	t.Run("vende sin stock si la empresa lo permite", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Sede")
		e.store.policies[e.actor.TenantID] = domain.StockPolicy{AllowNegative: true}
		p := newID()

		_, err := e.svc.IssueSaleStock(t.Context(), e.actor, app.SaleStockInput{
			LocationID: loc.ID(), SaleID: newID(), Lines: []app.SaleStockLine{saleLine(p, "1")},
		})
		require.NoError(t, err)
		requireDecimal(t, "-1", e.level(p, loc.ID()).Quantity)
	})
}

func TestReverseSaleStock(t *testing.T) {
	sell := func(t *testing.T, e *env, locationID, saleID uuid.UUID, lines ...app.SaleStockLine) {
		t.Helper()
		_, err := e.svc.IssueSaleStock(t.Context(), e.actor, app.SaleStockInput{LocationID: locationID, SaleID: saleID, Lines: lines})
		require.NoError(t, err)
	}

	t.Run("devuelve exactamente lo que salió en la venta", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Sede")
		p1, p2 := newID(), newID()
		e.receive(t, p1, loc.ID(), "10", "1000")
		e.receive(t, p2, loc.ID(), "5", "200")
		saleID := newID()
		sell(t, e, loc.ID(), saleID, saleLine(p1, "3"), saleLine(p2, "1.25"))
		sell(t, e, loc.ID(), newID(), saleLine(p1, "1"))
		// Entra mercancía más cara después de la venta.
		e.receive(t, p1, loc.ID(), "6", "1300")

		movements, err := e.svc.ReverseSaleStock(t.Context(), e.actor, saleID)
		require.NoError(t, err)

		require.Len(t, movements, 2)
		for _, m := range movements {
			require.Equal(t, domain.MovementSaleVoid, m.Type())
			require.Equal(t, &domain.DocumentRef{Kind: domain.DocumentSale, ID: saleID}, m.Reference())
			require.Equal(t, loc.ID(), m.LocationID())
		}
		requireDecimal(t, "15", e.level(p1, loc.ID()).Quantity) // 10 − 3 − 1 + 6 + 3
		requireDecimal(t, "5", e.level(p2, loc.ID()).Quantity)
		// La devolución entra al costo con que salió (1000), no al promedio vigente.
		byProduct := map[uuid.UUID]*domain.StockMovement{}
		for _, m := range movements {
			byProduct[m.ProductID()] = m
		}
		requireDecimal(t, "3", byProduct[p1].Quantity())
		requireDecimal(t, "1000", byProduct[p1].UnitCost())
		requireDecimal(t, "1.25", byProduct[p2].Quantity())

		require.Equal(t, int64(2), e.counter(t, "cuadra.inventory.movements", "type", "sale_void"))
		_, attrs := e.span(t, "inventory.ReverseSaleStock")
		require.Equal(t, saleID.String(), attrs["sale.id"].AsString())
	})

	t.Run("no devuelve dos veces el stock de la misma venta", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Sede")
		p := newID()
		e.receive(t, p, loc.ID(), "10", "1000")
		saleID := newID()
		sell(t, e, loc.ID(), saleID, saleLine(p, "4"))
		_, err := e.svc.ReverseSaleStock(t.Context(), e.actor, saleID)
		require.NoError(t, err)

		_, err = e.svc.ReverseSaleStock(t.Context(), e.actor, saleID)
		require.ErrorIs(t, err, domain.ErrSaleAlreadyReversed)
		requireDecimal(t, "10", e.level(p, loc.ID()).Quantity)
	})

	t.Run("una venta sin salidas no devuelve nada", func(t *testing.T) {
		e := newEnv(t)
		movements, err := e.svc.ReverseSaleStock(t.Context(), e.actor, newID())
		require.NoError(t, err)
		require.Empty(t, movements)
	})

	t.Run("no ve las salidas de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Sede")
		p := newID()
		e.receive(t, p, loc.ID(), "10", "1000")
		saleID := newID()
		sell(t, e, loc.ID(), saleID, saleLine(p, "4"))
		other := e.actor
		other.TenantID = newID()

		movements, err := e.svc.ReverseSaleStock(t.Context(), other, saleID)
		require.NoError(t, err)
		require.Empty(t, movements)
		requireDecimal(t, "6", e.level(p, loc.ID()).Quantity)
	})
}
