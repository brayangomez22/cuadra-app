package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

// registered is a purchase of two lines: a hardware store's screws and a
// clothing store's socks.
func (e *env) registered(t *testing.T, pay domain.Payment) *domain.Purchase {
	t.Helper()
	s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
	screws := e.variant("Tornillo drywall 6x1", "", "und", "0.19")
	socks := e.variant("Medias deportivas", "M / Blanco", "par", "0.19")
	p, err := e.svc.RegisterPurchase(t.Context(), e.actor,
		e.purchase(s.ID(), "FE-1", pay, line(screws, nil, "200", "450"), line(socks, nil, "36", "4500")))
	require.NoError(t, err)
	return p
}

func TestVoidPurchase(t *testing.T) {
	t.Run("anula la compra y saca su stock en la misma transacción", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCredit)

		voided, err := e.svc.VoidPurchase(t.Context(), e.actor, p.ID(), "Factura duplicada")
		require.NoError(t, err)

		require.Equal(t, domain.PurchaseVoided, voided.Status())
		require.Equal(t, e.actor.UserID, voided.Voided().By)
		require.Equal(t, []reversal{{PurchaseID: p.ID(), UserID: e.actor.UserID}}, e.stock.reversals)
		stored, err := e.svc.GetPurchase(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Equal(t, domain.PurchaseVoided, stored.Status())

		require.Equal(t, int64(1), e.counter(t, "cuadra.purchases.voided", "payment", "credit"))
		attrs := e.span(t, "purchases.VoidPurchase")
		require.Equal(t, p.ID().String(), attrs["purchase.id"].AsString())
	})

	t.Run("si ya se vendió parte del stock, no anula y nombra la línea", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCash)
		socks := p.Lines()[1].VariantID()
		e.stock.reverseErr = &app.VariantError{VariantID: socks, Err: domain.ErrInsufficientStock}

		_, err := e.svc.VoidPurchase(t.Context(), e.actor, p.ID(), "Error")

		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		var lineErr *app.LineError
		require.ErrorAs(t, err, &lineErr)
		require.Equal(t, 1, lineErr.Index)
		stored, err := e.svc.GetPurchase(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Equal(t, domain.PurchasePaid, stored.Status())
	})

	t.Run("no anula una compra con pagos ni toca el stock", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCredit)
		_, err := e.svc.RecordSupplierPayment(t.Context(), e.actor, p.ID(), app.SupplierPaymentRequest{Method: "cash", Amount: dec("1000")})
		require.NoError(t, err)

		_, err = e.svc.VoidPurchase(t.Context(), e.actor, p.ID(), "Error")
		require.ErrorIs(t, err, domain.ErrPurchaseHasPayments)
		require.Empty(t, e.stock.reversals)
	})

	t.Run("una compra de otra empresa no existe", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCash)
		other := app.Actor{TenantID: newID(), UserID: newID()}
		_, err := e.svc.VoidPurchase(t.Context(), other, p.ID(), "Error")
		require.ErrorIs(t, err, domain.ErrPurchaseNotFound)
	})

	t.Run("lista las compras anuladas por su estado", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCash)
		_, err := e.svc.VoidPurchase(t.Context(), e.actor, p.ID(), "Error")
		require.NoError(t, err)

		page, err := e.svc.ListPurchases(t.Context(), e.actor, app.PurchaseQuery{Status: domain.PurchaseVoided, Limit: 20})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		require.Equal(t, domain.PurchaseVoided, page.Items[0].Status())
	})
}
