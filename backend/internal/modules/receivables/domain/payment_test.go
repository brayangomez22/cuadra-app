package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func paymentInput(amount string) domain.PaymentInput {
	return domain.PaymentInput{
		ID: newID(), Method: domain.PaymentCash, Amount: dec(amount), UserID: newID(), At: soldAt.Add(time.Hour),
	}
}

func TestAccountCollect(t *testing.T) {
	t.Run("un abono parcial reduce el saldo sin dejarlo negativo", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		r := charge(t, a, "200000", soldAt)
		p, err := a.Collect(paymentInput("80000"), []*domain.Receivable{r})
		require.NoError(t, err)
		require.True(t, dec("80000").Equal(p.Amount()))
		require.True(t, dec("120000").Equal(r.Balance()))
		require.Equal(t, domain.ReceivableOpen, r.Status())
		require.True(t, dec("120000").Equal(a.Balance()))
		require.Equal(t, customerID, p.CustomerID())
	})

	t.Run("un abono total deja la cuenta pagada y el saldo en cero", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		r := charge(t, a, "200000", soldAt)
		_, err := a.Collect(paymentInput("200000"), []*domain.Receivable{r})
		require.NoError(t, err)
		require.True(t, r.Balance().IsZero())
		require.Equal(t, domain.ReceivablePaid, r.Status())
		require.True(t, a.Balance().IsZero())
	})

	t.Run("rechaza abono mayor al saldo y no cambia nada", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		r := charge(t, a, "200000", soldAt)
		_, err := a.Collect(paymentInput("200000.01"), []*domain.Receivable{r})
		require.ErrorIs(t, err, domain.ErrPaymentExceedsBalance)
		require.True(t, dec("200000").Equal(r.Balance()))
		require.True(t, dec("200000").Equal(a.Balance()))
	})

	t.Run("rechaza abono a un cliente que no debe nada", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		_, err := a.Collect(paymentInput("1000"), nil)
		require.ErrorIs(t, err, domain.ErrPaymentExceedsBalance)
	})

	t.Run("rechaza abono de monto cero, negativo o con más de dos decimales", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		r := charge(t, a, "200000", soldAt)
		for _, amount := range []string{"0", "-100", "1.005"} {
			_, err := a.Collect(paymentInput(amount), []*domain.Receivable{r})
			require.ErrorIs(t, err, domain.ErrInvalidPaymentAmount)
		}
	})

	t.Run("rechaza abono con medio de pago desconocido o sin usuario", func(t *testing.T) {
		a := newAccount(t, "500000", 30)
		r := charge(t, a, "200000", soldAt)
		in := paymentInput("1000")
		in.Method = "credit"
		_, err := a.Collect(in, []*domain.Receivable{r})
		require.ErrorIs(t, err, domain.ErrInvalidPaymentMethod)
		in = paymentInput("1000")
		in.UserID = uuid.Nil
		_, err = a.Collect(in, []*domain.Receivable{r})
		require.ErrorIs(t, err, domain.ErrInvalidUserID)
	})

	t.Run("el abono se aplica primero a la cuenta que vence antes", func(t *testing.T) {
		a := newAccount(t, "1000000", 30)
		newer := charge(t, a, "100000", soldAt.AddDate(0, 0, 5))
		older := charge(t, a, "100000", soldAt)
		p, err := a.Collect(paymentInput("150000"), []*domain.Receivable{newer, older})
		require.NoError(t, err)
		require.Equal(t, domain.ReceivablePaid, older.Status())
		require.True(t, dec("50000").Equal(newer.Balance()))
		require.Equal(t, []domain.Allocation{
			{ReceivableID: older.ID(), Amount: dec("100000")},
			{ReceivableID: newer.ID(), Amount: dec("50000")},
		}, normalize(p.Allocations()))
	})

	t.Run("acepta los medios de pago de un abono", func(t *testing.T) {
		for _, m := range []string{"cash", "card", "nequi", "daviplata", "transfer"} {
			got, err := domain.ParsePaymentMethod(m)
			require.NoError(t, err)
			require.Equal(t, m, got.String())
		}
		_, err := domain.ParsePaymentMethod("CASH")
		require.ErrorIs(t, err, domain.ErrInvalidPaymentMethod)
	})
}

// normalize rewrites the allocations' amounts so require.Equal compares
// their values, not their decimal representation.
func normalize(in []domain.Allocation) []domain.Allocation {
	out := make([]domain.Allocation, len(in))
	for i, a := range in {
		out[i] = domain.Allocation{ReceivableID: a.ReceivableID, Amount: dec(a.Amount.String())}
	}
	return out
}
