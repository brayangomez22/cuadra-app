package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func aPayment(amount string) domain.PaymentInput {
	return domain.PaymentInput{Method: domain.MethodTransfer, Amount: dec(amount), PaidOn: today, UserID: userID}
}

func TestPurchasePay(t *testing.T) {
	later := now.Add(time.Hour)

	t.Run("abona a una compra a crédito y baja su saldo", func(t *testing.T) {
		// 2 cajas a 45.000 + IVA 19 % = 107.100.
		p := newPurchase(t, domain.PaymentCredit, screwBoxes("2", "45000"))

		pay, err := p.Pay(aPayment("50000"), later)
		require.NoError(t, err)

		require.NotEqual(t, uuid.Nil, pay.ID())
		require.Equal(t, p.ID(), pay.PurchaseID())
		require.Equal(t, p.SupplierID(), pay.SupplierID())
		require.Equal(t, domain.MethodTransfer, pay.Method())
		require.True(t, dec("50000").Equal(pay.Amount()))
		require.Equal(t, today, pay.PaidOn())
		require.Equal(t, userID, pay.UserID())
		require.Equal(t, later, pay.CreatedAt())
		require.True(t, dec("57100").Equal(p.BalanceDue()), p.BalanceDue().String())
		require.True(t, dec("57100").Equal(pay.BalanceAfter()))
		require.Equal(t, domain.PurchaseOpen, p.Status())
	})

	t.Run("pagar todo el saldo deja la compra pagada", func(t *testing.T) {
		// 3 docenas a 54.000 + IVA 19 % = 192.780.
		p := newPurchase(t, domain.PaymentCredit, sockDozens("3", "54000"))
		_, err := p.Pay(aPayment(p.BalanceDue().String()), later)
		require.NoError(t, err)
		require.Equal(t, domain.PurchasePaid, p.Status())
	})

	t.Run("no deja el saldo negativo ni abona a una compra de contado", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCredit, screwBoxes("2", "45000"))
		_, err := p.Pay(aPayment("107100.01"), later)
		require.ErrorIs(t, err, domain.ErrPaymentExceedsBalance)
		require.True(t, dec("107100").Equal(p.BalanceDue()))

		cash := newPurchase(t, domain.PaymentCash, screwBoxes("1", "45000"))
		_, err = cash.Pay(aPayment("1"), later)
		require.ErrorIs(t, err, domain.ErrPaymentExceedsBalance)
	})

	t.Run("no abona a una compra anulada", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCredit, screwBoxes("2", "45000"))
		require.NoError(t, p.Void(userID, "Error", later))
		_, err := p.Pay(aPayment("1000"), later)
		require.ErrorIs(t, err, domain.ErrPurchaseVoided)
	})

	t.Run("rechaza montos, medios y fechas inválidos", func(t *testing.T) {
		cases := []struct {
			name string
			edit func(*domain.PaymentInput)
			want error
		}{
			{"monto cero", func(in *domain.PaymentInput) { in.Amount = dec("0") }, domain.ErrInvalidPaymentAmount},
			{"monto negativo", func(in *domain.PaymentInput) { in.Amount = dec("-5") }, domain.ErrInvalidPaymentAmount},
			{"monto con centavos de más", func(in *domain.PaymentInput) { in.Amount = dec("10.005") }, domain.ErrInvalidPaymentAmount},
			{"medio de pago desconocido", func(in *domain.PaymentInput) { in.Method = "credit" }, domain.ErrInvalidPaymentMethod},
			{"fecha futura", func(in *domain.PaymentInput) { in.PaidOn = today.AddDays(1) }, domain.ErrPaymentDateInFuture},
			{"antes de la factura", func(in *domain.PaymentInput) { in.PaidOn = today.AddDays(-30) }, domain.ErrPaymentBeforeInvoice},
			{"sin quién lo registra", func(in *domain.PaymentInput) { in.UserID = uuid.Nil }, domain.ErrInvalidUserID},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				p := newPurchase(t, domain.PaymentCredit, screwBoxes("2", "45000"))
				in := aPayment("1000")
				tc.edit(&in)
				_, err := p.Pay(in, later)
				require.ErrorIs(t, err, tc.want)
				require.True(t, dec("107100").Equal(p.BalanceDue()))
			})
		}
	})

	t.Run("los medios de pago se leen por su código", func(t *testing.T) {
		for _, code := range []string{"cash", "card", "nequi", "daviplata", "transfer"} {
			m, err := domain.ParsePaymentMethod(code)
			require.NoError(t, err)
			require.Equal(t, code, string(m))
		}
		_, err := domain.ParsePaymentMethod("credit")
		require.ErrorIs(t, err, domain.ErrInvalidPaymentMethod)
	})

	t.Run("un pago guardado se reconstruye igual", func(t *testing.T) {
		p := newPurchase(t, domain.PaymentCredit, screwBoxes("2", "45000"))
		pay, err := p.Pay(aPayment("7100"), later)
		require.NoError(t, err)

		back, err := domain.RehydrateSupplierPayment(pay.Snapshot())
		require.NoError(t, err)
		require.Equal(t, pay, back)
	})
}

func TestPayablesWindow(t *testing.T) {
	t.Run("lo vencido es antes de hoy y lo que vence pronto, hasta dentro de 7 días", func(t *testing.T) {
		w := domain.PayablesWindowOn(today)
		require.Equal(t, today, w.OverdueBefore)
		require.Equal(t, today.AddDays(7), w.DueSoonUntil)
	})
}
