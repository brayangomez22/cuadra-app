package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func TestRecordPayment(t *testing.T) {
	t.Run("el abono en efectivo entra a la caja abierta y reduce el saldo", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente de ferretería", "500000", 30)
		e.charge(t, id, "200000")
		res, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "cash", Amount: dec("80000")})
		require.NoError(t, err)
		require.Equal(t, &e.drawer.session, res.Payment.CashSessionID())
		require.True(t, dec("120000").Equal(res.Account.Balance()))
		require.Len(t, e.store.cashIns, 1)
		require.True(t, dec("80000").Equal(e.store.cashIns[0]))
		require.Equal(t, []string{"Abono de cartera"}, e.drawer.reasons)
		attrs := e.span(t, "receivables.RecordPayment")
		require.Equal(t, id.String(), attrs["customer.id"].AsString())
		require.Equal(t, res.Payment.ID().String(), attrs["payment.id"].AsString())
		require.Equal(t, int64(1), e.counter(t, "cuadra.receivables.payments", "payment.method", "cash"))
	})

	t.Run("el abono por transferencia no toca la caja", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Clienta de almacén de ropa", "500000", 15)
		e.charge(t, id, "90000")
		res, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "transfer", Amount: dec("90000")})
		require.NoError(t, err)
		require.Nil(t, res.Payment.CashSessionID())
		require.Empty(t, e.store.cashIns)
		require.True(t, res.Account.Balance().IsZero())
	})

	t.Run("sin caja abierta el abono en efectivo se acepta si el tenant no exige caja", func(t *testing.T) {
		e := newEnv(t)
		e.drawer.closed = true
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		e.charge(t, id, "50000")
		res, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "cash", Amount: dec("50000")})
		require.NoError(t, err)
		require.Nil(t, res.Payment.CashSessionID())
	})

	t.Run("rechaza abono en efectivo sin caja abierta si el tenant la exige", func(t *testing.T) {
		e := newEnv(t)
		e.drawer.closed, e.drawer.required = true, true
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		e.charge(t, id, "50000")
		_, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "cash", Amount: dec("50000")})
		require.ErrorIs(t, err, domain.ErrCashSessionRequired)
	})

	t.Run("rechaza abono mayor al saldo y no deja efectivo en la caja", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		e.charge(t, id, "50000")
		_, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "cash", Amount: dec("50000.01")})
		require.ErrorIs(t, err, domain.ErrPaymentExceedsBalance)
		require.Empty(t, e.store.cashIns)
		view, err := e.svc.CustomerCredit(t.Context(), e.actor, id)
		require.NoError(t, err)
		require.True(t, dec("50000").Equal(view.Account.Balance()))
	})

	t.Run("rechaza abono con medio de pago inválido antes de tocar la caja", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		e.charge(t, id, "50000")
		_, err := e.svc.RecordPayment(t.Context(), e.actor, id, app.PaymentRequest{Method: "credit", Amount: dec("100")})
		require.ErrorIs(t, err, domain.ErrInvalidPaymentMethod)
		require.Empty(t, e.store.cashIns)
	})

	t.Run("rechaza abono de un cliente que no existe", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.RecordPayment(t.Context(), e.actor, uuid.Must(uuid.NewV7()), app.PaymentRequest{Method: "cash", Amount: dec("100")})
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})
}
