//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// tenantDefaultTaxRateAdded is 00019_tenant_default_tax_rate: the IVA rate
// each tenant proposes to new products.
const tenantDefaultTaxRateAdded = 19

func TestTenantDefaultTaxRateMigration(t *testing.T) {
	t.Run("los tenants existentes quedan con IVA por defecto de 19 %", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantBusinessTemplateAdded)
		tenants := seedTemplateTenants(t, s, 3)

		s.MigrateTo(t, tenantDefaultTaxRateAdded)

		rows, err := connect(t, s.SuperuserURL).Query(t.Context(), "SELECT id, default_tax_rate::text FROM tenants")
		require.NoError(t, err)
		got := map[uuid.UUID]string{}
		for rows.Next() {
			var id uuid.UUID
			var rate string
			require.NoError(t, rows.Scan(&id, &rate))
			got[id] = rate
		}
		require.NoError(t, rows.Err())
		require.Equal(t, map[uuid.UUID]string{tenants[0]: "0.1900", tenants[1]: "0.1900", tenants[2]: "0.1900"}, got)
	})

	t.Run("un tenant nuevo debe traer su IVA por defecto, de 0 a menos de 1", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantDefaultTaxRateAdded)
		conn := connect(t, s.SuperuserURL)
		insert := func(rate string) error {
			_, err := conn.Exec(t.Context(), `
				INSERT INTO tenants (id, name, nit, status, business_template, default_tax_rate, created_at)
				VALUES ($1, 'Negocio', '900123456-7', 'active', 'generica', `+rate+`, now())`, uuid.Must(uuid.NewV7()))
			return err
		}

		for _, rate := range []string{"NULL", "-0.01", "1", "1.19"} {
			require.Error(t, insert(rate), rate)
		}
		for _, rate := range []string{"0", "0.05", "0.19"} {
			require.NoError(t, insert(rate), rate)
		}
	})

	t.Run("el Down quita la columna", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantBusinessTemplateAdded)
		seedTemplateTenants(t, s, 1)
		s.MigrateTo(t, tenantDefaultTaxRateAdded)

		s.MigrateDownTo(t, tenantBusinessTemplateAdded)

		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(), `
			SELECT EXISTS (SELECT 1 FROM information_schema.columns
			               WHERE table_name = 'tenants' AND column_name = 'default_tax_rate')`).Scan(&exists))
		require.False(t, exists)
	})
}

// seedTemplateTenants inserts n tenants as they were at 00018, with their
// business template, as the superuser (the tenants table has RLS).
func seedTemplateTenants(t *testing.T, s *dbtest.Server, n int) []uuid.UUID {
	t.Helper()
	conn := connect(t, s.SuperuserURL)
	ids := make([]uuid.UUID, 0, n)
	for range n {
		id := uuid.Must(uuid.NewV7())
		_, err := conn.Exec(t.Context(), `
			INSERT INTO tenants (id, name, nit, status, business_template, created_at)
			VALUES ($1, 'Negocio', '900123456-7', 'active', 'ferreteria', now())`, id)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	return ids
}
