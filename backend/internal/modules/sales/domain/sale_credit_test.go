package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// customerSale is a sale to a customer with one line of 20000 + 19% IVA
// (total 23800).
func customerSale(t *testing.T) *domain.Sale {
	t.Helper()
	customerID := newID()
	s, err := domain.NewSale(domain.NewSaleParams{
		ID: newID(), TenantID: newID(), LocationID: newID(), UserID: newID(), CustomerID: &customerID, CreatedAt: testNow,
	})
	require.NoError(t, err)
	addLine(t, s, line("2", "10000", "0.19", ""))
	return s
}

func TestSaleOnCredit(t *testing.T) {
	t.Run("acepta crédito como medio de pago", func(t *testing.T) {
		got, err := domain.ParsePaymentMethod("credit")
		require.NoError(t, err)
		require.Equal(t, domain.PaymentCredit, got)
	})

	t.Run("una venta toda a crédito queda confirmada sin cambio y con su monto a crédito", func(t *testing.T) {
		s := customerSale(t)
		require.NoError(t, s.Confirm([]domain.Payment{payment(t, domain.PaymentCredit, "23800")}, testNow))
		require.Equal(t, domain.SaleConfirmed, s.Status())
		requireDec(t, "23800", s.CreditAmount())
		require.True(t, s.Change().IsZero())
	})

	t.Run("una venta con abono en efectivo y el resto a crédito", func(t *testing.T) {
		s := customerSale(t)
		require.NoError(t, s.Confirm([]domain.Payment{
			payment(t, domain.PaymentCash, "10000"),
			payment(t, domain.PaymentCredit, "13800"),
		}, testNow))
		requireDec(t, "13800", s.CreditAmount())
	})

	t.Run("el pago a crédito no da cambio", func(t *testing.T) {
		s := customerSale(t)
		err := s.Confirm([]domain.Payment{payment(t, domain.PaymentCredit, "30000")}, testNow)
		require.ErrorIs(t, err, domain.ErrNonCashOverpayment)
		require.Equal(t, domain.SaleDraft, s.Status())
	})

	t.Run("rechaza venta a crédito sin cliente", func(t *testing.T) {
		s := newSale(t)
		addLine(t, s, line("1", "1000", "0", ""))
		err := s.Confirm([]domain.Payment{payment(t, domain.PaymentCredit, "1000")}, testNow)
		require.ErrorIs(t, err, domain.ErrCreditWithoutCustomer)
		require.Equal(t, domain.SaleDraft, s.Status())
	})

	t.Run("rechaza dos pagos a crédito en una venta", func(t *testing.T) {
		s := customerSale(t)
		err := s.Confirm([]domain.Payment{
			payment(t, domain.PaymentCredit, "10000"),
			payment(t, domain.PaymentCredit, "13800"),
		}, testNow)
		require.ErrorIs(t, err, domain.ErrDuplicateCreditPayment)
	})

	t.Run("una venta de contado no tiene monto a crédito", func(t *testing.T) {
		s := customerSale(t)
		require.NoError(t, s.Confirm([]domain.Payment{payment(t, domain.PaymentCash, "23800")}, testNow))
		require.True(t, s.CreditAmount().IsZero())
	})
}
