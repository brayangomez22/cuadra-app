package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func TestCreditTerms(t *testing.T) {
	t.Run("un cliente sin cuenta no tiene crédito ni debe nada", func(t *testing.T) {
		e := newEnv(t)
		id := e.customer("Consumidor de ferretería")
		view, err := e.svc.CustomerCredit(t.Context(), e.actor, id)
		require.NoError(t, err)
		require.False(t, view.Account.Terms().Enabled())
		require.True(t, view.Account.Balance().IsZero())
		require.Empty(t, view.Open)
	})

	t.Run("asigna cupo y plazo a clientes de distintos tipos de negocio", func(t *testing.T) {
		cases := []struct {
			name, limit string
			days        int
		}{
			{"cliente frecuente de ferretería", "3000000", 30},
			{"clienta de almacén de ropa", "400000", 15},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				e := newEnv(t)
				id := e.customer(c.name)
				a, err := e.svc.SetCreditTerms(t.Context(), e.actor, id, app.TermsRequest{Limit: dec(c.limit), Days: c.days})
				require.NoError(t, err)
				require.True(t, dec(c.limit).Equal(a.Terms().Limit()))
				view, err := e.svc.CustomerCredit(t.Context(), e.actor, id)
				require.NoError(t, err)
				require.True(t, dec(c.limit).Equal(view.Account.Available()))
				require.Equal(t, c.days, view.Account.Terms().Days())
				require.Equal(t, id.String(), e.span(t, "receivables.SetCreditTerms")["customer.id"].AsString())
			})
		}
	})

	t.Run("cambiar el cupo conserva el saldo", func(t *testing.T) {
		e := newEnv(t)
		id := e.creditCustomer(t, "Cliente", "500000", 30)
		e.charge(t, id, "200000")
		_, err := e.svc.SetCreditTerms(t.Context(), e.actor, id, app.TermsRequest{Limit: dec("100000"), Days: 10})
		require.NoError(t, err)
		view, err := e.svc.CustomerCredit(t.Context(), e.actor, id)
		require.NoError(t, err)
		require.True(t, dec("200000").Equal(view.Account.Balance()))
		require.True(t, view.Account.Available().IsZero())
		require.Len(t, view.Open, 1)
	})

	t.Run("rechaza cupo de un cliente que no existe", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SetCreditTerms(t.Context(), e.actor, newID(), app.TermsRequest{Limit: dec("1000"), Days: 30})
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
		_, err = e.svc.CustomerCredit(t.Context(), e.actor, newID())
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("rechaza cupo o plazo inválidos", func(t *testing.T) {
		e := newEnv(t)
		id := e.customer("Cliente")
		_, err := e.svc.SetCreditTerms(t.Context(), e.actor, id, app.TermsRequest{Limit: dec("-1"), Days: 30})
		require.ErrorIs(t, err, domain.ErrInvalidCreditLimit)
		_, err = e.svc.SetCreditTerms(t.Context(), e.actor, id, app.TermsRequest{Limit: dec("1000"), Days: 400})
		require.ErrorIs(t, err, domain.ErrInvalidCreditTerm)
	})
}
