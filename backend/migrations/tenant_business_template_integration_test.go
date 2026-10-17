//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// tenantBusinessTemplateAdded is 00018_tenant_business_template: the
// business template each tenant signed up with.
const tenantBusinessTemplateAdded = 18

func TestTenantBusinessTemplateMigration(t *testing.T) {
	t.Run("los tenants existentes quedan con la plantilla ferreteria y conservan sus módulos", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantModulesAdded)
		tenants := seedTenants(t, s, 3)
		conn := connect(t, s.SuperuserURL)
		_, err := conn.Exec(t.Context(),
			"INSERT INTO tenant_modules (tenant_id, module_id, enabled_at) VALUES ($1, 'variants', now())", tenants[0])
		require.NoError(t, err)

		s.MigrateTo(t, tenantBusinessTemplateAdded)

		rows, err := conn.Query(t.Context(), "SELECT id, business_template FROM tenants")
		require.NoError(t, err)
		got := map[uuid.UUID]string{}
		for rows.Next() {
			var id uuid.UUID
			var template string
			require.NoError(t, rows.Scan(&id, &template))
			got[id] = template
		}
		require.NoError(t, rows.Err())
		require.Equal(t, map[uuid.UUID]string{tenants[0]: "ferreteria", tenants[1]: "ferreteria", tenants[2]: "ferreteria"}, got)

		var modules int
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT count(*) FROM tenant_modules WHERE tenant_id = $1 AND module_id = 'variants'", tenants[0]).Scan(&modules))
		require.Equal(t, 1, modules, "the migration does not turn variants off")
	})

	t.Run("un tenant nuevo debe tener plantilla y con formato de ID", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantBusinessTemplateAdded)
		conn := connect(t, s.SuperuserURL)

		for _, template := range []string{"NULL", "''", "'Ferreteria'", "'ropa-calzado'"} {
			_, err := conn.Exec(t.Context(),
				"INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Negocio', '900123456-7', 'active', "+template+", now())",
				uuid.Must(uuid.NewV7()))
			require.Error(t, err, template)
		}
	})

	t.Run("la migración restaura FORCE en tenants", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantModulesAdded)
		seedTenants(t, s, 1)

		s.MigrateTo(t, tenantBusinessTemplateAdded)

		var enabled, forced bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE relname = 'tenants'").Scan(&enabled, &forced))
		require.True(t, enabled)
		require.True(t, forced, "FORCE is restored on tenants")
	})

	t.Run("el Down quita la columna", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantModulesAdded)
		seedTenants(t, s, 1)
		s.MigrateTo(t, tenantBusinessTemplateAdded)

		s.MigrateDownTo(t, tenantModulesAdded)

		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(), `
			SELECT EXISTS (SELECT 1 FROM information_schema.columns
			               WHERE table_name = 'tenants' AND column_name = 'business_template')`).Scan(&exists))
		require.False(t, exists)
	})

	t.Run("la migración falla si los conteos no cuadran", func(t *testing.T) {
		s := dbtest.StartAt(t, tenantModulesAdded)
		tenants := seedTenants(t, s, 2)
		// Simulates a backfill that silently skips a tenant (as a FORCE ROW
		// LEVEL SECURITY left on would): a trigger (test only) drops the
		// update of one tenant.
		_, err := connect(t, s.SuperuserURL).Exec(t.Context(), `
			CREATE FUNCTION test_skip_one_tenant() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF NEW.id = '`+tenants[1].String()+`' THEN RETURN NULL; END IF;
				RETURN NEW;
			END $$;
			CREATE TRIGGER test_skip BEFORE UPDATE ON tenants
				FOR EACH ROW EXECUTE FUNCTION test_skip_one_tenant();`)
		require.NoError(t, err)

		err = s.TryMigrateTo(t, tenantBusinessTemplateAdded)

		require.ErrorContains(t, err, "tenants backfill: 2 tenants but 1 have a business template")
	})
}
