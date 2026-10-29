//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func TestCreditRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("un cliente sin cuenta tiene cupo cero; el cupo y el plazo se guardan", func(t *testing.T) {
		tn := f.createTenant(t)
		c := f.createCustomer(t, tn, "Cliente de ferretería")
		a, err := f.credit.Account(t.Context(), tn.id, c)
		require.NoError(t, err)
		require.False(t, a.Terms().Enabled())

		f.setTerms(t, tn, c, "2000000", 30)
		f.setTerms(t, tn, c, "1500000.50", 45)
		a, err = f.credit.Account(t.Context(), tn.id, c)
		require.NoError(t, err)
		require.True(t, dec("1500000.50").Equal(a.Terms().Limit()))
		require.Equal(t, 45, a.Terms().Days())
	})

	t.Run("rechaza el cupo de un cliente que no existe", func(t *testing.T) {
		tn := f.createTenant(t)
		terms, err := domain.NewTerms(dec("1000"), 30)
		require.NoError(t, err)
		a, err := domain.NewAccount(tn.id, newID(), terms)
		require.NoError(t, err)
		require.ErrorIs(t, f.credit.SaveTerms(t.Context(), a, now), domain.ErrCustomerNotFound)
	})

	t.Run("una venta a crédito y un abono dejan el saldo igual a la suma de las cuentas abiertas", func(t *testing.T) {
		tn := f.createTenant(t)
		c := f.createCustomer(t, tn, "Clienta de almacén de ropa")
		f.setTerms(t, tn, c, "500000", 15)
		r := f.charge(t, tn, c, "120000")
		f.charge(t, tn, c, "30000")
		require.NoError(t, f.pay(t, tn, c, "130000"))

		a, err := f.credit.Account(t.Context(), tn.id, c)
		require.NoError(t, err)
		require.True(t, dec("20000").Equal(a.Balance()), a.Balance().String())
		require.True(t, a.Balance().Equal(f.openSum(t, tn, c)))

		of, err := f.credit.OfSale(t.Context(), tn.id, r.SaleID())
		require.NoError(t, err)
		require.Equal(t, domain.ReceivablePaid, of.Status)
		require.Equal(t, domain.NewDate(2026, 11, 12), r.DueDate())
	})

	t.Run("si la operación falla no guarda nada", func(t *testing.T) {
		tn := f.createTenant(t)
		c := f.createCustomer(t, tn, "Cliente")
		f.setTerms(t, tn, c, "500000", 30)
		f.charge(t, tn, c, "100000")
		require.ErrorIs(t, f.pay(t, tn, c, "100000.01"), domain.ErrPaymentExceedsBalance)
		require.True(t, dec("100000").Equal(f.openSum(t, tn, c)))
	})

	t.Run("el estado de cuenta trae cargos, abonos y anulaciones", func(t *testing.T) {
		tn := f.createTenant(t)
		c := f.createCustomer(t, tn, "Cliente")
		f.setTerms(t, tn, c, "500000", 30)
		f.charge(t, tn, c, "100000")
		canceled := f.charge(t, tn, c, "20000")
		require.NoError(t, f.credit.Update(t.Context(), tn.id, c, func(_ context.Context, a *domain.Account, open []*domain.Receivable) (domain.Change, error) {
			for _, r := range open {
				if r.ID() == canceled.ID() {
					return domain.Change{Changed: []*domain.Receivable{r}}, a.Cancel(r, now)
				}
			}
			return domain.Change{}, errors.New("not found")
		}))
		require.NoError(t, f.pay(t, tn, c, "40000"))

		movements, err := f.credit.Movements(t.Context(), tn.id, c)
		require.NoError(t, err)
		statement := domain.Statement(movements)
		require.Len(t, statement, 4)
		require.Equal(t, domain.MovementPayment, statement[0].Kind)
		require.True(t, dec("60000").Equal(statement[0].Balance), statement[0].Balance.String())
		require.Equal(t, domain.PaymentTransfer, statement[0].Method)
	})

	t.Run("dos ventas simultáneas no pasan juntas el cupo", func(t *testing.T) {
		tn := f.createTenant(t)
		c := f.createCustomer(t, tn, "Cliente")
		f.setTerms(t, tn, c, "100000", 30)
		sales := []string{"70000", "70000"}
		errs := make([]error, len(sales))
		var wg sync.WaitGroup
		for i, amount := range sales {
			saleID := f.createSale(t, tn, amount)
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[i] = f.credit.Update(context.Background(), tn.id, c, func(_ context.Context, a *domain.Account, _ []*domain.Receivable) (domain.Change, error) {
					r, err := a.Charge(domain.ChargeInput{ID: newID(), SaleID: saleID, Amount: dec(amount), At: now})
					return domain.Change{Opened: r}, err
				})
			}()
		}
		wg.Wait()
		rejected := 0
		for _, err := range errs {
			if errors.Is(err, domain.ErrCreditLimitExceeded) {
				rejected++
			} else {
				require.NoError(t, err)
			}
		}
		require.Equal(t, 1, rejected)
		require.True(t, dec("70000").Equal(f.openSum(t, tn, c)))
	})

	t.Run("la cartera abierta trae saldo y vencimiento de cada cuenta", func(t *testing.T) {
		tn := f.createTenant(t)
		c := f.createCustomer(t, tn, "Cliente")
		f.setTerms(t, tn, c, "500000", 0)
		f.charge(t, tn, c, "50000")
		balances, err := f.credit.OpenBalances(t.Context(), tn.id)
		require.NoError(t, err)
		require.Len(t, balances, 1)
		require.Equal(t, domain.NewDate(2026, 10, 28), balances[0].DueDate)
		require.True(t, dec("50000").Equal(balances[0].Balance))
	})

	t.Run("aislamiento: el tenant B no ve ni modifica la cartera del A", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		c := f.createCustomer(t, a, "Cliente de A")
		f.setTerms(t, a, c, "500000", 30)
		r := f.charge(t, a, c, "100000")

		account, err := f.credit.Account(t.Context(), b.id, c)
		require.NoError(t, err)
		require.False(t, account.Terms().Enabled(), "B no ve la cuenta de A")
		open, err := f.credit.Open(t.Context(), b.id, c)
		require.NoError(t, err)
		require.Empty(t, open)
		balances, err := f.credit.OpenBalances(t.Context(), b.id)
		require.NoError(t, err)
		require.Empty(t, balances)
		_, err = f.credit.OfSale(t.Context(), b.id, r.SaleID())
		require.ErrorIs(t, err, domain.ErrReceivableNotFound)
		movements, err := f.credit.Movements(t.Context(), b.id, c)
		require.NoError(t, err)
		require.Empty(t, movements)

		require.ErrorIs(t, f.pay(t, b, c, "100"), domain.ErrPaymentExceedsBalance, "B no abona a la cuenta de A")
		require.True(t, dec("100000").Equal(f.openSum(t, a, c)))
	})
}
