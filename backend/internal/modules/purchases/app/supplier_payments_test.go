package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestRecordSupplierPayment(t *testing.T) {
	t.Run("abona a una compra a crédito, hoy en Bogotá si no dice la fecha", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCredit)
		owed := p.BalanceDue()

		res, err := e.svc.RecordSupplierPayment(t.Context(), e.actor, p.ID(), app.SupplierPaymentRequest{
			Method: "transfer", Amount: dec("100000"),
		})
		require.NoError(t, err)

		require.Equal(t, today, res.Payment.PaidOn())
		require.Equal(t, domain.MethodTransfer, res.Payment.Method())
		require.True(t, owed.Sub(dec("100000")).Equal(res.Purchase.BalanceDue()))
		payments, err := e.svc.PurchasePayments(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.Len(t, payments, 1)
		require.Equal(t, res.Payment.ID(), payments[0].ID())

		require.Equal(t, int64(1), e.counter(t, "cuadra.purchases.supplier_payments", "method", "transfer"))
		attrs := e.span(t, "purchases.RecordSupplierPayment")
		require.Equal(t, p.ID().String(), attrs["purchase.id"].AsString())
		require.Equal(t, res.Payment.ID().String(), attrs["payment.id"].AsString())
	})

	t.Run("respeta la fecha del pago y no deja el saldo negativo", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCredit)
		yesterday := today.AddDays(-1)

		res, err := e.svc.RecordSupplierPayment(t.Context(), e.actor, p.ID(), app.SupplierPaymentRequest{
			Method: "cash", Amount: p.BalanceDue(), PaidOn: &yesterday,
		})
		require.NoError(t, err)
		require.Equal(t, yesterday, res.Payment.PaidOn())
		require.Equal(t, domain.PurchasePaid, res.Purchase.Status())

		_, err = e.svc.RecordSupplierPayment(t.Context(), e.actor, p.ID(), app.SupplierPaymentRequest{Method: "cash", Amount: dec("1")})
		require.ErrorIs(t, err, domain.ErrPaymentExceedsBalance)
	})

	t.Run("rechaza un medio de pago desconocido", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCredit)
		_, err := e.svc.RecordSupplierPayment(t.Context(), e.actor, p.ID(), app.SupplierPaymentRequest{Method: "credit", Amount: dec("1")})
		require.ErrorIs(t, err, domain.ErrInvalidPaymentMethod)
	})

	t.Run("los pagos de una compra de otra empresa no existen", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCredit)
		_, err := e.svc.PurchasePayments(t.Context(), app.Actor{TenantID: newID(), UserID: newID()}, p.ID())
		require.ErrorIs(t, err, domain.ErrPurchaseNotFound)
	})
}

func TestListPayables(t *testing.T) {
	t.Run("pide lo que se debe con la ventana de hoy en Bogotá", func(t *testing.T) {
		e := newEnv(t)
		page, err := e.svc.ListPayables(t.Context(), e.actor, app.PayablesQuery{Limit: 20})
		require.NoError(t, err)
		require.Equal(t, domain.PayablesWindowOn(today), e.purchases.window)
		require.Empty(t, page.Items)
		e.span(t, "purchases.ListPayables")
	})
}
