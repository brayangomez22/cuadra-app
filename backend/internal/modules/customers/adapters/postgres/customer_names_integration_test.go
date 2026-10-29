//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCustomerNames(t *testing.T) {
	f := newFixture(t)

	t.Run("devuelve los nombres de los clientes pedidos, activos o no", func(t *testing.T) {
		tenant := f.createTenant(t)
		hardware := f.createCustomer(t, tenant, company("900123456-8", "Construcciones del Valle SAS", nil))
		inactive := person("1020304050", "Laura Restrepo")
		inactive.Active = false
		clothing := f.createCustomer(t, tenant, inactive)
		f.createCustomer(t, tenant, person("79845123", "No pedido"))

		names, err := f.customers.Names(t.Context(), tenant, []uuid.UUID{hardware.ID(), clothing.ID(), uuid.Must(uuid.NewV7())})
		require.NoError(t, err)
		require.Equal(t, map[uuid.UUID]string{
			hardware.ID(): "Construcciones del Valle SAS", clothing.ID(): "Laura Restrepo",
		}, names)
	})

	t.Run("aislamiento: el tenant B no ve los nombres de los clientes del A", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		c := f.createCustomer(t, a, person("1020304050", "Carlos Pérez"))
		names, err := f.customers.Names(t.Context(), b, []uuid.UUID{c.ID()})
		require.NoError(t, err)
		require.Empty(t, names)
	})
}
