package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestConfirmSaleDocument(t *testing.T) {
	t.Run("pide el documento electrónico de la venta de contado sin cliente (ferretería)", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		require.Equal(t, []app.DocumentRequest{{
			TenantID: e.actor.TenantID, UserID: e.actor.UserID, SaleID: sale.ID(),
		}}, e.documents.requested)
	})

	t.Run("pide factura para la venta a crédito con cliente y el tipo que pidió (almacén de ropa)", func(t *testing.T) {
		e := newEnv(t)
		in := e.creditSale(t, "0", "11900")
		in.DocumentKind = "invoice"
		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, in)
		require.NoError(t, err)
		require.Equal(t, []app.DocumentRequest{{
			TenantID: e.actor.TenantID, UserID: e.actor.UserID, SaleID: sale.ID(),
			Kind: "invoice", Credit: true, HasCustomer: true,
		}}, e.documents.requested)
	})

	t.Run("si el documento no se puede numerar la venta no se guarda", func(t *testing.T) {
		e := newEnv(t)
		e.documents.err = domain.ErrNoActiveResolution
		p := e.addProduct("Camiseta básica", "10000", "0.19", true)
		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(),
			Lines:      []app.LineRequest{lineReq(p, "1", "0")},
			Payments:   []app.PaymentRequest{cash("11900")},
		})
		require.ErrorIs(t, err, domain.ErrNoActiveResolution)
		require.Zero(t, e.storedSales())
	})
}

func TestVoidSaleWithDocument(t *testing.T) {
	t.Run("no anula una venta con documento electrónico", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		e.documents.invoiced[sale.ID()] = true
		_, err := e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Error del cajero")
		require.ErrorIs(t, err, domain.ErrSaleInvoiced)
		got, err := e.svc.GetSale(t.Context(), e.actor, sale.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleConfirmed, got.Status())
		require.Empty(t, e.store.reversed, "no devuelve el stock")
	})

	t.Run("anula una venta sin documento electrónico", func(t *testing.T) {
		e := newEnv(t)
		sale := e.sell(t, newID())
		_, err := e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Error del cajero")
		require.NoError(t, err)
	})
}
