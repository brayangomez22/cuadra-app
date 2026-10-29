//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// demoTenantsAdded is 00027_demo_tenants: the demo flag and the functions
// that find and purge the demo tenants.
const demoTenantsAdded = 27

// seedTenant stores a tenant of template (demo or not) with one user and
// one customer segment, as the owner within its tenant.
func seedTenant(t *testing.T, s *dbtest.Server, template string, demo bool) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	conn := connect(t, s.AdminURL)
	tx, err := conn.Begin(t.Context())
	require.NoError(t, err)
	exec := func(sql string, args ...any) {
		_, err := tx.Exec(t.Context(), sql, args...)
		require.NoError(t, err)
	}
	exec("SELECT set_config('app.tenant_id', $1, true)", id.String())
	exec(`INSERT INTO tenants (id, name, nit, status, business_template, demo, created_at)
		VALUES ($1, 'Negocio', '900123456-8', 'active', $2, $3, now())`, id, template, demo)
	exec(`INSERT INTO users (id, tenant_id, email, name, password_hash, role, active, created_at)
		VALUES ($1, $2, 'ana@negocio.test', 'Ana', 'hash', 'owner', true, now())`, uuid.Must(uuid.NewV7()), id)
	exec(`INSERT INTO customer_segments (id, tenant_id, name, created_at) VALUES ($1, $2, 'Mayorista', now())`,
		uuid.Must(uuid.NewV7()), id)
	require.NoError(t, tx.Commit(t.Context()))
	return id
}

// rowsOf counts the rows of tenant in every table with a tenant_id, and its
// tenants row, as the superuser (which bypasses Row-Level Security).
func rowsOf(t *testing.T, conn *pgx.Conn, tenant uuid.UUID) int {
	t.Helper()
	total := 0
	for _, table := range tenantTables(t, conn) {
		var n int
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()+" WHERE tenant_id = $1", tenant).Scan(&n))
		total += n
	}
	var n int
	require.NoError(t, conn.QueryRow(t.Context(), "SELECT count(*) FROM tenants WHERE id = $1", tenant).Scan(&n))
	return total + n
}

// tenantTables are the tables of the public schema with a tenant_id column.
func tenantTables(t *testing.T, conn *pgx.Conn) []string {
	t.Helper()
	rows, err := conn.Query(t.Context(), `
		SELECT c.relname FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid
		WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND a.attname = 'tenant_id' AND NOT a.attisdropped
		ORDER BY c.relname`)
	require.NoError(t, err)
	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	return tables
}

func purge(t *testing.T, s *dbtest.Server, tenant uuid.UUID) string {
	t.Helper()
	var result string
	require.NoError(t, connect(t, s.AppURL).QueryRow(t.Context(), "SELECT purge_demo_tenant($1)", tenant).Scan(&result))
	return result
}

func TestDemoTenantsMigration(t *testing.T) {
	s := dbtest.Start(t)
	super := connect(t, s.SuperuserURL)

	t.Run("toda tabla con tenant_id deja leer y borrar a demo_purge", func(t *testing.T) {
		tables := tenantTables(t, super)
		require.Contains(t, tables, "stock_movements")
		for _, table := range append(tables, "tenants") {
			var canSelect, canDelete bool
			require.NoError(t, super.QueryRow(t.Context(),
				"SELECT has_table_privilege('demo_purge', $1, 'SELECT'), has_table_privilege('demo_purge', $1, 'DELETE')",
				"public."+table).Scan(&canSelect, &canDelete))
			require.True(t, canSelect && canDelete, "demo_purge needs SELECT and DELETE on %s", table)
		}
	})

	t.Run("borra el tenant demo con todas sus filas", func(t *testing.T) {
		for _, template := range []string{"ferreteria", "ropa_calzado"} {
			demo := seedTenant(t, s, template, true)
			require.Equal(t, 3, rowsOf(t, super, demo))

			require.Equal(t, "purged", purge(t, s, demo))
			require.Zero(t, rowsOf(t, super, demo), template)
		}
	})

	t.Run("aislamiento: borrar el demo no toca otro tenant", func(t *testing.T) {
		demo := seedTenant(t, s, "ferreteria", true)
		other := seedTenant(t, s, "ferreteria", false)

		require.Equal(t, "purged", purge(t, s, demo))
		require.Equal(t, 3, rowsOf(t, super, other))
	})

	t.Run("no borra un tenant que no es demo ni uno que no existe", func(t *testing.T) {
		kept := seedTenant(t, s, "ropa_calzado", false)

		require.Equal(t, "not_demo", purge(t, s, kept))
		require.Equal(t, 3, rowsOf(t, super, kept))
		require.Equal(t, "not_found", purge(t, s, uuid.Must(uuid.NewV7())))
	})

	t.Run("demo_tenants devuelve solo los demo de la plantilla", func(t *testing.T) {
		wanted := seedTenant(t, s, "papeleria", true)
		seedTenant(t, s, "papeleria", false)
		seedTenant(t, s, "tienda", true)

		rows, err := connect(t, s.AppURL).Query(t.Context(), "SELECT demo_tenants('papeleria')")
		require.NoError(t, err)
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{wanted}, ids)
	})

	t.Run("app_user no ve los otros tenants demo fuera de la función", func(t *testing.T) {
		seedTenant(t, s, "generica", true)
		var n int
		require.NoError(t, connect(t, s.AppURL).QueryRow(t.Context(), "SELECT count(*) FROM tenants").Scan(&n))
		require.Zero(t, n)
	})
}

func TestDemoTenantsMigrationDown(t *testing.T) {
	t.Run("se revierte y se vuelve a aplicar", func(t *testing.T) {
		s := dbtest.StartAt(t, demoTenantsAdded)
		seedTenant(t, s, "ferreteria", true)

		s.MigrateDownTo(t, demoTenantsAdded-1)
		s.MigrateTo(t, demoTenantsAdded)
	})
}
