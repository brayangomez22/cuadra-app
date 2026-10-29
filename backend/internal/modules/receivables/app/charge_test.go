package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func TestCharge(t *testing.T) {
	t.Run("la venta a crédito abre la cuenta por cobrar con vencimiento en el día de Bogotá", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente de ferretería", "500000", 30)
		saleID := newID()
		r, err := e.svc.Charge(t.Context(), e.actor, app.ChargeRequest{CustomerID: id, SaleID: saleID, Amount: dec("120000")})
		require.NoError(t, err)
		require.Equal(t, domain.NewDate(2026, 11, 27), r.DueDate())
		view, err := e.svc.CustomerCredit(t.Context(), e.actor, id)
		require.NoError(t, err)
		require.True(t, dec("380000").Equal(view.Account.Available()))
		attrs := e.span(t, "receivables.Charge")
		require.Equal(t, saleID.String(), attrs["sale.id"].AsString())
		require.Equal(t, id.String(), attrs["customer.id"].AsString())
		require.Equal(t, int64(1), e.counter(t, "cuadra.receivables.charges", "credit.authorized", "false"))
	})

	t.Run("un cliente sin cupo no puede comprar a crédito", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.Charge(t.Context(), e.actor, app.ChargeRequest{CustomerID: e.customer("Sin cupo"), SaleID: newID(), Amount: dec("1000")})
		require.ErrorIs(t, err, domain.ErrCreditNotEnabled)
	})

	t.Run("rechaza venta que excede el cupo sin autorización y la cuenta en la métrica", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Clienta de almacén de ropa", "300000", 15)
		e.charge(t, id, "250000")
		_, err := e.svc.Charge(t.Context(), e.actor, app.ChargeRequest{CustomerID: id, SaleID: newID(), Amount: dec("60000")})
		var exceeded *domain.CreditLimitExceededError
		require.ErrorAs(t, err, &exceeded)
		require.True(t, dec("50000").Equal(exceeded.Available))
		require.Equal(t, int64(1), e.counter(t, "cuadra.receivables.limit_rejections", "", ""))
	})

	t.Run("acepta venta que excede el cupo con autorización y la cuenta como autorizada", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "100000", 30)
		authorizer := newID()
		r, err := e.svc.Charge(t.Context(), e.actor, app.ChargeRequest{
			CustomerID: id, SaleID: newID(), Amount: dec("150000"), AuthorizedBy: &authorizer,
		})
		require.NoError(t, err)
		require.Equal(t, authorizer, r.Authorization().UserID)
		require.Equal(t, e.now, r.Authorization().At)
		require.Equal(t, int64(1), e.counter(t, "cuadra.receivables.charges", "credit.authorized", "true"))
	})
}

func TestCancelSale(t *testing.T) {
	t.Run("anular una venta a crédito sin abonos cancela su cuenta y libera el cupo", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		r := e.charge(t, id, "200000")
		require.NoError(t, e.svc.CancelSale(t.Context(), e.actor, r.SaleID()))
		view, err := e.svc.CustomerCredit(t.Context(), e.actor, id)
		require.NoError(t, err)
		require.True(t, view.Account.Balance().IsZero())
		require.Empty(t, view.Open)
		require.Equal(t, r.SaleID().String(), e.span(t, "receivables.CancelSale")["sale.id"].AsString())
		require.Equal(t, int64(1), e.counter(t, "cuadra.receivables.cancellations", "", ""))
	})

	t.Run("rechaza anular una venta a crédito con abonos", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		r := e.charge(t, id, "200000")
		_, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "transfer", Amount: dec("1000")})
		require.NoError(t, err)
		require.ErrorIs(t, e.svc.CancelSale(t.Context(), e.actor, r.SaleID()), domain.ErrReceivableHasPayments)
	})

	t.Run("rechaza anular una venta a crédito ya pagada", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		r := e.charge(t, id, "200000")
		_, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "card", Amount: dec("200000")})
		require.NoError(t, err)
		require.ErrorIs(t, e.svc.CancelSale(t.Context(), e.actor, r.SaleID()), domain.ErrReceivableHasPayments)
	})

	t.Run("rechaza anular una venta que no abrió cuenta por cobrar", func(t *testing.T) {
		e := newEnv(t)
		require.ErrorIs(t, e.svc.CancelSale(t.Context(), e.actor, newID()), domain.ErrReceivableNotFound)
	})
}
