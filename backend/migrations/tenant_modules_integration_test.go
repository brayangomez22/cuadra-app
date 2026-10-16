//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// tenantModulesAdded is 00017_tenant_modules: modules activable per tenant.
const tenantModulesAdded = 17

// seedTenants stores n tenants before 00017, as the schema owner inside a
// transaction scoped to each tenant (the way RLS lets the owner write).
func seedTenants(t *testing.T, s *dbtest.Server, n int) []uuid.UUID {
	t.Helper()
	conn := connect(t, s.AdminURL)
	ids := make([]uuid.UUID, 0, n)
	for range n {
		id := uuid.Must(uuid.NewV7())
		tx, err := conn.Begin(t.Context())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(), "SELECT set_config('app.tenant_id', $1, true)", id.String())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Negocio', '900123456-7', 'active', $2)", id, created)
		require.NoError(t, err)
		require.NoError(t, tx.Commit(t.Context()))
		ids = append(ids, id)
	}
	return ids
}

func TestTenantModulesMigration(t *testing.T) {
	t.Run("los tenants existentes quedan con el módulo variants activo", func(t *testing.T) {
		s := dbtest.StartAt(t, axesByTenant)
		tenants := seedTenants(t, s, 3)

		s.MigrateTo(t, tenantModulesAdded)

		rows, err := connect(t, s.SuperuserURL).Query(t.Context(),
			"SELECT tenant_id, module_id, enabled_by IS NULL FROM tenant_modules")
		require.NoError(t, err)
		got := map[uuid.UUID]string{}
		for rows.Next() {
			var tenant uuid.UUID
			var module string
			var byMigration bool
			require.NoError(t, rows.Scan(&tenant, &module, &byMigration))
			require.True(t, byMigration, "no user enabled it")
			got[tenant] = module
		}
		require.NoError(t, rows.Err())
		require.Equal(t, map[uuid.UUID]string{tenants[0]: "variants", tenants[1]: "variants", tenants[2]: "variants"}, got)
	})

	t.Run("la migración restaura FORCE en las tablas que toca", func(t *testing.T) {
		s := dbtest.StartAt(t, axesByTenant)
		seedTenants(t, s, 1)

		s.MigrateTo(t, tenantModulesAdded)

		conn := connect(t, s.SuperuserURL)
		for _, table := range []string{"tenants", "tenant_modules"} {
			var enabled, forced bool
			require.NoError(t, conn.QueryRow(t.Context(),
				"SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE relname = $1", table).Scan(&enabled, &forced))
			require.True(t, enabled, "RLS is enabled on %s", table)
			require.True(t, forced, "FORCE is restored on %s", table)
		}
	})

	t.Run("el Down quita la tabla", func(t *testing.T) {
		s := dbtest.StartAt(t, axesByTenant)
		seedTenants(t, s, 1)
		s.MigrateTo(t, tenantModulesAdded)

		s.MigrateDownTo(t, axesByTenant)

		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT to_regclass('public.tenant_modules') IS NOT NULL").Scan(&exists))
		require.False(t, exists)
	})

	t.Run("la migración falla si los conteos no cuadran", func(t *testing.T) {
		s := dbtest.StartAt(t, axesByTenant)
		tenants := seedTenants(t, s, 2)
		// Simulates a backfill that silently stores fewer rows (as a FORCE
		// ROW LEVEL SECURITY left on would): when 00017 creates the table, an
		// event trigger (superuser only, test only) attaches a trigger that
		// drops the modules of one tenant.
		_, err := connect(t, s.SuperuserURL).Exec(t.Context(), `
			CREATE FUNCTION test_drop_one_tenant() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF NEW.tenant_id = '`+tenants[1].String()+`' THEN RETURN NULL; END IF;
				RETURN NEW;
			END $$;
			CREATE FUNCTION test_attach_drop() RETURNS event_trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF EXISTS (SELECT 1 FROM pg_event_trigger_ddl_commands()
				           WHERE command_tag = 'CREATE TABLE' AND object_identity = 'public.tenant_modules') THEN
					CREATE TRIGGER test_drop BEFORE INSERT ON public.tenant_modules
						FOR EACH ROW EXECUTE FUNCTION test_drop_one_tenant();
				END IF;
			END $$;
			CREATE EVENT TRIGGER test_attach_drop ON ddl_command_end EXECUTE FUNCTION test_attach_drop();`)
		require.NoError(t, err)

		err = s.TryMigrateTo(t, tenantModulesAdded)

		require.ErrorContains(t, err, "tenant_modules backfill: 2 tenants but 1 have the variants module")
		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT to_regclass('public.tenant_modules') IS NOT NULL").Scan(&exists))
		require.False(t, exists, "the migration rolled back")
	})
}
