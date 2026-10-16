//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// Versions of the units migration under test.
const (
	beforeUnits   = 13 // 00013_sale_lines_by_variant: products.base_unit is a code of a fixed list
	unitsByTenant = 14 // 00014_units_of_measure
)

// defaultUnits are the units of the fixed list, by code, with the name each
// tenant gets.
var defaultUnits = map[string]string{
	"und": "Unidad", "m": "Metro", "kg": "Kilogramo", "l": "Litro",
	"caja": "Caja", "rollo": "Rollo", "bulto": "Bulto", "galon": "Galón",
}

// unitsSeed is the data stored before 00014: a hardware store (tenantA), a
// clothing store (tenantB) and a tenant without products (empty), with the
// base unit code of each product.
type unitsSeed struct {
	tenantA, tenantB, empty uuid.UUID
	productUnit             map[uuid.UUID]string
	productTenant           map[uuid.UUID]uuid.UUID
}

func seedUnitCodes(t *testing.T, s *dbtest.Server) unitsSeed {
	t.Helper()
	newID := func() uuid.UUID { return uuid.Must(uuid.NewV7()) }
	seed := unitsSeed{
		tenantA: newID(), tenantB: newID(), empty: newID(),
		productUnit: map[uuid.UUID]string{}, productTenant: map[uuid.UUID]uuid.UUID{},
	}
	conn := connect(t, s.AdminURL)
	inTenant := func(tenant uuid.UUID, fn func(tx pgx.Tx)) {
		tx, err := conn.Begin(t.Context())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(), "SELECT set_config('app.tenant_id', $1, true)", tenant.String())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Negocio', '900123456-7', 'active', $2)", tenant, created)
		require.NoError(t, err)
		fn(tx)
		require.NoError(t, tx.Commit(t.Context()))
	}
	product := func(tx pgx.Tx, tenant uuid.UUID, name, unit string) {
		id := newID()
		_, err := tx.Exec(t.Context(), `INSERT INTO products (id, tenant_id, name, description, base_unit, tax_rate, active, created_at)
			VALUES ($1, $2, $3, '', $4, 0.19, true, $5)`, id, tenant, name, unit, created)
		require.NoError(t, err)
		seed.productUnit[id], seed.productTenant[id] = unit, tenant
	}
	inTenant(seed.tenantA, func(tx pgx.Tx) {
		product(tx, seed.tenantA, "Cemento gris 50 kg", "bulto")
		product(tx, seed.tenantA, "Cable 12 AWG", "m")
		product(tx, seed.tenantA, "Pintura blanca", "galon")
		product(tx, seed.tenantA, "Tornillo 1/4", "und")
	})
	inTenant(seed.tenantB, func(tx pgx.Tx) {
		product(tx, seed.tenantB, "Camiseta básica", "und")
		product(tx, seed.tenantB, "Tela por metro", "m")
	})
	inTenant(seed.empty, func(pgx.Tx) {})
	return seed
}

func TestUnitsOfMeasureMigration(t *testing.T) {
	t.Run("cada tenant existente queda con las unidades por defecto", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeUnits)
		seed := seedUnitCodes(t, s)

		s.MigrateTo(t, unitsByTenant)

		conn := connect(t, s.SuperuserURL)
		for _, tenant := range []uuid.UUID{seed.tenantA, seed.tenantB, seed.empty} {
			rows, err := conn.Query(t.Context(), "SELECT id, code, name FROM units_of_measure WHERE tenant_id = $1", tenant)
			require.NoError(t, err)
			got := map[string]string{}
			for rows.Next() {
				var id uuid.UUID
				var code, name string
				require.NoError(t, rows.Scan(&id, &code, &name))
				require.Equal(t, 7, int(id.Version()), code)
				got[code] = name
			}
			require.NoError(t, rows.Err())
			require.Equal(t, defaultUnits, got)
		}
	})

	t.Run("la migración asigna a cada producto la unidad de su tenant con el mismo código", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeUnits)
		seed := seedUnitCodes(t, s)

		s.MigrateTo(t, unitsByTenant)

		conn := connect(t, s.SuperuserURL)
		for id, code := range seed.productUnit {
			var unitTenant uuid.UUID
			var unitCode string
			require.NoError(t, conn.QueryRow(t.Context(), `
				SELECT u.tenant_id, u.code FROM products p JOIN units_of_measure u ON u.id = p.base_unit_id
				WHERE p.id = $1`, id).Scan(&unitTenant, &unitCode))
			require.Equal(t, seed.productTenant[id], unitTenant, code)
			require.Equal(t, code, unitCode)
		}

		var left int
		require.NoError(t, conn.QueryRow(t.Context(), `
			SELECT count(*) FROM information_schema.columns
			WHERE table_name = 'products' AND column_name = 'base_unit'`).Scan(&left))
		require.Zero(t, left, "base_unit left products")
		for _, table := range []string{"units_of_measure", "products", "tenants"} {
			var forced bool
			require.NoError(t, conn.QueryRow(t.Context(),
				"SELECT relforcerowsecurity FROM pg_class WHERE relname = $1", table).Scan(&forced))
			require.True(t, forced, "FORCE is restored on %s", table)
		}
	})

	t.Run("una unidad en uso no se puede borrar", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeUnits)
		seed := seedUnitCodes(t, s)
		s.MigrateTo(t, unitsByTenant)

		_, err := connect(t, s.SuperuserURL).Exec(t.Context(),
			"DELETE FROM units_of_measure WHERE tenant_id = $1 AND code = 'bulto'", seed.tenantA)

		require.ErrorContains(t, err, "products_base_unit_fkey")
	})

	t.Run("el Down restaura base_unit desde el código de la unidad", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeUnits)
		seed := seedUnitCodes(t, s)
		s.MigrateTo(t, unitsByTenant)

		s.MigrateDownTo(t, beforeUnits)

		conn := connect(t, s.SuperuserURL)
		for id, code := range seed.productUnit {
			var got string
			require.NoError(t, conn.QueryRow(t.Context(), "SELECT base_unit FROM products WHERE id = $1", id).Scan(&got))
			require.Equal(t, code, got)
		}
		var exists bool
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT to_regclass('public.units_of_measure') IS NOT NULL").Scan(&exists))
		require.False(t, exists)
		var forced bool
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT relforcerowsecurity FROM pg_class WHERE relname = 'products'").Scan(&forced))
		require.True(t, forced)
	})

	t.Run("el Down falla si un producto usa una unidad que no estaba en la lista fija", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeUnits)
		seed := seedUnitCodes(t, s)
		s.MigrateTo(t, unitsByTenant)
		_, err := connect(t, s.SuperuserURL).Exec(t.Context(), `
			WITH dozen AS (
				INSERT INTO units_of_measure (id, tenant_id, code, name, created_at)
				VALUES ($1, $2, 'docena', 'Docena', now()) RETURNING id
			)
			UPDATE products SET base_unit_id = (SELECT id FROM dozen) WHERE tenant_id = $2`,
			uuid.Must(uuid.NewV7()), seed.tenantB)
		require.NoError(t, err)

		err = s.TryMigrateDownTo(t, beforeUnits)

		require.ErrorContains(t, err, "units_of_measure restore: 2 products use a unit outside the fixed list")
	})

	t.Run("la migración falla si los conteos no cuadran", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeUnits)
		seed := seedUnitCodes(t, s)
		// Simulates a seed that silently stores fewer rows (as a FORCE ROW
		// LEVEL SECURITY left on would): when 00014 creates the table, an
		// event trigger (superuser only, test only) attaches a trigger that
		// drops the units of one tenant.
		_, err := connect(t, s.SuperuserURL).Exec(t.Context(), `
			CREATE FUNCTION test_drop_one_tenant() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF NEW.tenant_id = '`+seed.empty.String()+`' THEN RETURN NULL; END IF;
				RETURN NEW;
			END $$;
			CREATE FUNCTION test_attach_drop() RETURNS event_trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF EXISTS (SELECT 1 FROM pg_event_trigger_ddl_commands()
				           WHERE command_tag = 'CREATE TABLE' AND object_identity = 'public.units_of_measure') THEN
					CREATE TRIGGER test_drop BEFORE INSERT ON public.units_of_measure
						FOR EACH ROW EXECUTE FUNCTION test_drop_one_tenant();
				END IF;
			END $$;
			CREATE EVENT TRIGGER test_attach_drop ON ddl_command_end EXECUTE FUNCTION test_attach_drop();`)
		require.NoError(t, err)

		err = s.TryMigrateTo(t, unitsByTenant)

		require.ErrorContains(t, err, "units_of_measure backfill: 3 tenants need 24 units but 16 were stored")
		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT to_regclass('public.units_of_measure') IS NOT NULL").Scan(&exists))
		require.False(t, exists, "the migration rolled back")
	})
}
