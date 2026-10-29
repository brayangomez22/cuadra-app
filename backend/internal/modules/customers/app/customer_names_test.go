package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCustomerNames(t *testing.T) {
	t.Run("devuelve el nombre de los clientes pedidos y omite los que no son del tenant", func(t *testing.T) {
		e := newEnv(t)
		hardware := e.addCustomer(t, company("900123456-8", "Construcciones del Valle SAS"))
		clothing := e.addCustomer(t, person("1020304050", "Laura Restrepo"))
		e.addCustomer(t, person("79845123", "No pedido"))

		names, err := e.svc.CustomerNames(t.Context(), e.actor, []uuid.UUID{hardware.ID(), clothing.ID(), newID()})
		require.NoError(t, err)
		require.Equal(t, map[uuid.UUID]string{
			hardware.ID(): "Construcciones del Valle SAS",
			clothing.ID(): "Laura Restrepo",
		}, names)
		e.span(t, "customers.CustomerNames")
	})

	t.Run("sin ids no consulta nada", func(t *testing.T) {
		e := newEnv(t)
		names, err := e.svc.CustomerNames(t.Context(), e.actor, nil)
		require.NoError(t, err)
		require.Empty(t, names)
	})
}
