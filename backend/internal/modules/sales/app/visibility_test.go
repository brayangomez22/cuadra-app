package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSalesVisibility(t *testing.T) {
	t.Run("el cajero solo ve sus ventas del día", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ListSales(t.Context(), e.actor, app.ListSalesInput{Limit: 20})
		require.NoError(t, err)
		require.Equal(t, &e.actor.UserID, e.store.filters[0].CashierID)
	})

	t.Run("quien ve todas las ventas no filtra por cajero", func(t *testing.T) {
		e := newEnv(t)
		manager := e.actor
		manager.SeesAllSales = true
		_, err := e.svc.ListSales(t.Context(), manager, app.ListSalesInput{Limit: 20})
		require.NoError(t, err)
		require.Nil(t, e.store.filters[0].CashierID)
	})

	t.Run("el cajero no abre la venta de otro cajero, el administrador sí", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		otherCashier := e.actor
		otherCashier.UserID = newID()
		_, err := e.svc.GetSale(t.Context(), otherCashier, sale.ID())
		require.ErrorIs(t, err, domain.ErrSaleNotFound)

		manager := otherCashier
		manager.SeesAllSales = true
		_, err = e.svc.GetSale(t.Context(), manager, sale.ID())
		require.NoError(t, err)
	})
}

func TestSaleDetail(t *testing.T) {
	t.Run("trae el nombre de quien vendió y de quien anuló", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		manager := e.actor
		manager.UserID, manager.SeesAllSales = newID(), true
		e.users.names[e.actor.UserID] = "Ana Cajera"
		e.users.names[manager.UserID] = "Luis Administrador"
		_, err := e.svc.VoidSale(t.Context(), manager, sale.ID(), "Error del cajero")
		require.NoError(t, err)

		detail, err := e.svc.GetSaleDetail(t.Context(), manager, sale.ID())
		require.NoError(t, err)
		require.Equal(t, sale.ID(), detail.Sale.ID())
		require.Equal(t, "Ana Cajera", detail.CashierName)
		require.Equal(t, "Luis Administrador", detail.VoidedByName)
	})

	t.Run("un usuario borrado deja el nombre vacío", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		detail, err := e.svc.GetSaleDetail(t.Context(), e.actor, sale.ID())
		require.NoError(t, err)
		require.Empty(t, detail.CashierName)
		require.Empty(t, detail.VoidedByName)
	})
}
