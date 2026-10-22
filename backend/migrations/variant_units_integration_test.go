//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// variantUnitsAdded is 00020_variant_units: alternate units of a variant,
// the unit of each sale line and the unit_conversions module.
const variantUnitsAdded = 20

// seedTenantWith stores a tenant of template before 00020.
func seedTenantWith(t *testing.T, s *dbtest.Server, template string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	_, err := connect(t, s.SuperuserURL).Exec(t.Context(), `
		INSERT INTO tenants (id, name, nit, status, business_template, created_at)
		VALUES ($1, 'Negocio', '900123456-7', 'active', $2, now())`, id, template)
	require.NoError(t, err)
	return id
}

func TestVariantUnitsMigration(t *testing.T) {
	t.Run("activa unit_conversions solo a los tenants cuya plantilla lo incluye", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantDefaultTaxRateAdded)
		hardware := seedTenantWith(t, s, "ferreteria")
		store := seedTenantWith(t, s, "tienda")
		generic := seedTenantWith(t, s, "generica")
		seedTenantWith(t, s, "ropa_calzado")
		seedTenantWith(t, s, "drogueria")

		s.MigrateTo(t, variantUnitsAdded)

		rows, err := connect(t, s.SuperuserURL).Query(t.Context(),
			"SELECT tenant_id FROM tenant_modules WHERE module_id = 'unit_conversions' AND enabled_by IS NULL")
		require.NoError(t, err)
		var got []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			require.NoError(t, rows.Scan(&id))
			got = append(got, id)
		}
		require.NoError(t, rows.Err())
		require.ElementsMatch(t, []uuid.UUID{hardware, store, generic}, got)
	})

	t.Run("la migración restaura FORCE y activa RLS en la tabla nueva", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantDefaultTaxRateAdded)
		seedTenantWith(t, s, "ferreteria")

		s.MigrateTo(t, variantUnitsAdded)

		conn := connect(t, s.SuperuserURL)
		for _, table := range []string{"tenants", "tenant_modules", "variant_units"} {
			var enabled, forced bool
			require.NoError(t, conn.QueryRow(t.Context(),
				"SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE relname = $1", table).Scan(&enabled, &forced))
			require.True(t, enabled, "RLS is enabled on %s", table)
			require.True(t, forced, "FORCE is restored on %s", table)
		}
	})

	t.Run("el Down quita la tabla, las columnas de la línea y el módulo", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantDefaultTaxRateAdded)
		seedTenantWith(t, s, "ferreteria")
		s.MigrateTo(t, variantUnitsAdded)

		s.MigrateDownTo(t, tenantDefaultTaxRateAdded)

		conn := connect(t, s.SuperuserURL)
		var table, column bool
		var modules int
		require.NoError(t, conn.QueryRow(t.Context(), `
			SELECT to_regclass('public.variant_units') IS NOT NULL,
			       EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sale_lines' AND column_name = 'unit_code'),
			       (SELECT count(*) FROM tenant_modules WHERE module_id = 'unit_conversions')`).Scan(&table, &column, &modules))
		require.False(t, table)
		require.False(t, column)
		require.Zero(t, modules)
	})
}
