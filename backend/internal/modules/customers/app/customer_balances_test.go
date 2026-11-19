package app_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
)

func TestListCustomersWithBalances(t *testing.T) {
	t.Run("con la cartera trae lo que debe cada cliente de la página", func(t *testing.T) {
		e := newEnv(t)
		hardware := e.addCustomer(t, company("900123456-8", "Constructora Andina")).ID()
		e.addCustomer(t, person("1020304050", "Clienta de almacén de ropa"))
		e.balances.owed[hardware] = app.Balance{Balance: decimal.RequireFromString("800000"), Overdue: decimal.RequireFromString("120000")}

		page, err := e.svc.ListCustomers(t.Context(), e.actor, app.CustomerQuery{Limit: 20, WithBalances: true})

		require.NoError(t, err)
		require.Len(t, e.balances.asked, 2, "pregunta por los de la página")
		b := page.Balances[hardware]
		require.True(t, decimal.RequireFromString("120000").Equal(b.Overdue))
		require.Len(t, page.Balances, 1)
	})
	t.Run("sin la cartera no pregunta lo que deben", func(t *testing.T) {
		e := newEnv(t)
		e.addCustomer(t, person("1020304050", "Carlos Pérez"))
		page, err := e.svc.ListCustomers(t.Context(), e.actor, app.CustomerQuery{Limit: 20})
		require.NoError(t, err)
		require.Nil(t, e.balances.asked)
		require.Nil(t, page.Balances)
	})
}
