//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestCustomerRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lee un cliente con todos sus datos", func(t *testing.T) {
		tenant := f.createTenant(t)
		segment := f.createSegment(t, tenant, "Mayorista").ID()
		c := f.createCustomer(t, tenant, company("900123456-8", "Confecciones La Moda S.A.S.", &segment))

		got, err := f.customers.GetByID(t.Context(), tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, c, got)

		minimal := f.createCustomer(t, tenant, person("1020304050", "Carlos Pérez"))
		got, err = f.customers.GetByID(t.Context(), tenant, minimal.ID())
		require.NoError(t, err)
		require.Equal(t, minimal, got)
	})

	t.Run("actualiza los datos y el estado", func(t *testing.T) {
		tenant := f.createTenant(t)
		c := f.createCustomer(t, tenant, person("1020304050", "Carlos Pérez"))
		p := person("1020304050", "Carlos Andrés Pérez")
		p.Active = false
		require.NoError(t, c.Update(p))

		require.NoError(t, f.customers.Update(t.Context(), c))

		got, err := f.customers.GetByID(t.Context(), tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, "Carlos Andrés Pérez", got.Name())
		require.False(t, got.Active())
	})

	t.Run("no permite dos clientes con el mismo documento en el tenant y dice cuál", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createCustomer(t, tenant, person("1020304050", "Carlos Pérez"))
		dup, err := domain.NewCustomer(tenant, person("1020304050", "Otro"), now)
		require.NoError(t, err)

		err = f.customers.Create(t.Context(), dup)

		var taken *domain.DocumentTakenError
		require.ErrorAs(t, err, &taken)
		require.Equal(t, "CC 1020304050", taken.Document.String())

		other := f.createCustomer(t, tenant, person("79845123", "Ana Gómez"))
		require.NoError(t, other.Update(person("1020304050", "Ana Gómez")))
		require.ErrorIs(t, f.customers.Update(t.Context(), other), domain.ErrDocumentTaken)
	})

	t.Run("el mismo número con otro tipo de documento no choca", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createCustomer(t, tenant, person("1020304050", "Carlos Pérez"))
		ce := person("1020304050", "Carlos Pérez")
		ce.DocumentType = domain.DocumentCE
		f.createCustomer(t, tenant, ce)
	})

	t.Run("rechaza un segmento de otro tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		foreign := f.createSegment(t, f.createTenant(t), "Mayorista").ID()
		c, err := domain.NewCustomer(tenant, company("900123456-8", "La Moda", &foreign), now)
		require.NoError(t, err)

		require.ErrorIs(t, f.customers.Create(t.Context(), c), domain.ErrSegmentNotFound)
	})

	t.Run("un cliente inexistente no se encuentra ni se actualiza", func(t *testing.T) {
		tenant := f.createTenant(t)
		c, err := domain.NewCustomer(tenant, person("1020304050", "Carlos Pérez"), now)
		require.NoError(t, err)

		_, err = f.customers.GetByID(t.Context(), tenant, c.ID())
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
		require.ErrorIs(t, f.customers.Update(t.Context(), c), domain.ErrCustomerNotFound)
	})

	t.Run("aislamiento: el tenant B no ve ni modifica los clientes del A", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		c := f.createCustomer(t, tenantA, person("1020304050", "Carlos Pérez"))

		_, err := f.customers.GetByID(t.Context(), tenantB, c.ID())
		require.ErrorIs(t, err, domain.ErrCustomerNotFound)
		items, total, err := f.customers.Search(t.Context(), domain.CustomerFilter{TenantID: tenantB, Status: domain.StatusAll, Limit: 20})
		require.NoError(t, err)
		require.Zero(t, total)
		require.Empty(t, items)

		snap := c.Snapshot()
		snap.TenantID = tenantB
		hijack, err := domain.RehydrateCustomer(snap)
		require.NoError(t, err)
		require.ErrorIs(t, f.customers.Update(t.Context(), hijack), domain.ErrCustomerNotFound)
		got, err := f.customers.GetByID(t.Context(), tenantA, c.ID())
		require.NoError(t, err)
		require.Equal(t, c, got)

		// Tenant B can register the same document: uniqueness is per tenant.
		f.createCustomer(t, tenantB, person("1020304050", "Carlos Pérez"))
	})
}
