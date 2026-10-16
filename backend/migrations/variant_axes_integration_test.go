//go:build integration

package migrations_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// Versions of the attribute and axis migrations under test.
const (
	beforeAttributes = 14 // 00014_units_of_measure
	attributesAdded  = 15 // 00015_product_attributes
	axesByTenant     = 16 // 00016_variant_axes: axes defined by each tenant
)

// axesSeed is the data stored before 00016: a clothing store (clothes) whose
// variants spell a size "M" and "m", a hardware store (hardware) with a
// presentation axis and a simple product, and a tenant without products
// (empty). axisValues holds every variant's stored axis values.
type axesSeed struct {
	clothes, hardware, empty uuid.UUID
	axisValues               map[uuid.UUID]map[string]string
}

func seedFreeAxes(t *testing.T, s *dbtest.Server) axesSeed {
	t.Helper()
	newID := func() uuid.UUID { return uuid.Must(uuid.NewV7()) }
	seed := axesSeed{clothes: newID(), hardware: newID(), empty: newID(), axisValues: map[uuid.UUID]map[string]string{}}
	conn := connect(t, s.AdminURL)
	inTenant := func(tenant uuid.UUID, fn func(tx pgx.Tx, unit uuid.UUID)) {
		tx, err := conn.Begin(t.Context())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(), "SELECT set_config('app.tenant_id', $1, true)", tenant.String())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Negocio', '900123456-7', 'active', $2)", tenant, created)
		require.NoError(t, err)
		unit := newID()
		_, err = tx.Exec(t.Context(),
			"INSERT INTO units_of_measure (id, tenant_id, code, name, created_at) VALUES ($1, $2, 'und', 'Unidad', $3)", unit, tenant, created)
		require.NoError(t, err)
		fn(tx, unit)
		require.NoError(t, tx.Commit(t.Context()))
	}
	product := func(tx pgx.Tx, tenant, unit uuid.UUID, name string, variants ...map[string]string) {
		id := newID()
		_, err := tx.Exec(t.Context(), `INSERT INTO products (id, tenant_id, name, description, base_unit_id, tax_rate, active, created_at)
			VALUES ($1, $2, $3, '', $4, 0.19, true, $5)`, id, tenant, name, unit, created)
		require.NoError(t, err)
		for i, axes := range variants {
			variantID := newID()
			raw, err := json.Marshal(axes)
			require.NoError(t, err)
			_, err = tx.Exec(t.Context(), `INSERT INTO product_variants
				(id, tenant_id, product_id, sku, cost, price, axis_values, is_default, active, created_at)
				VALUES ($1, $2, $3, $4, 1000, 2000, $5, $6, true, $7)`,
				variantID, tenant, id, strings.ToUpper(variantID.String()), raw, i == 0, created)
			require.NoError(t, err)
			seed.axisValues[variantID] = axes
		}
	}
	inTenant(seed.clothes, func(tx pgx.Tx, unit uuid.UUID) {
		product(tx, seed.clothes, unit, "Camiseta básica",
			map[string]string{"talla": "S", "color": "Rojo"}, map[string]string{"talla": "M", "color": "Rojo"})
		product(tx, seed.clothes, unit, "Pantalón", map[string]string{"talla": "m", "color": "Azul"})
	})
	inTenant(seed.hardware, func(tx pgx.Tx, unit uuid.UUID) {
		product(tx, seed.hardware, unit, "Pintura blanca",
			map[string]string{"presentación": "Galón"}, map[string]string{"presentación": "Cuarto"})
		product(tx, seed.hardware, unit, "Tornillo 1/4", map[string]string{})
	})
	inTenant(seed.empty, func(pgx.Tx, uuid.UUID) {})
	return seed
}

// axesOf returns tenant's axes with their values, read as the superuser.
func axesOf(t *testing.T, s *dbtest.Server, tenant uuid.UUID) map[string][]string {
	t.Helper()
	rows, err := connect(t, s.SuperuserURL).Query(t.Context(),
		"SELECT id, name, allowed_values FROM variant_axes WHERE tenant_id = $1", tenant)
	require.NoError(t, err)
	got := map[string][]string{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		var values []string
		require.NoError(t, rows.Scan(&id, &name, &values))
		require.Equal(t, 7, int(id.Version()), name)
		got[name] = values
	}
	require.NoError(t, rows.Err())
	return got
}

func TestProductAttributesMigration(t *testing.T) {
	t.Run("los productos existentes quedan sin atributos", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeAttributes)
		seedFreeAxes(t, s)

		s.MigrateTo(t, attributesAdded)

		var products, empty int
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT count(*), count(*) FILTER (WHERE attributes = '{}'::jsonb) FROM products").Scan(&products, &empty))
		require.Equal(t, 4, products)
		require.Equal(t, products, empty)
	})

	t.Run("el Down quita la columna y la tabla", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeAttributes)
		seedFreeAxes(t, s)
		s.MigrateTo(t, attributesAdded)

		s.MigrateDownTo(t, beforeAttributes)

		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT to_regclass('public.attribute_definitions') IS NOT NULL").Scan(&exists))
		require.False(t, exists)
	})
}

func TestVariantAxesMigration(t *testing.T) {
	t.Run("cada eje usado por las variantes queda definido en su tenant con todos sus valores", func(t *testing.T) {
		s := dbtest.StartAt(t, attributesAdded)
		seed := seedFreeAxes(t, s)

		s.MigrateTo(t, axesByTenant)

		// "M" and "m" stay as one value, with the first spelling in byte order.
		require.Equal(t, map[string][]string{"color": {"Azul", "Rojo"}, "talla": {"M", "S"}}, axesOf(t, s, seed.clothes))
		require.Equal(t, map[string][]string{"presentación": {"Cuarto", "Galón"}}, axesOf(t, s, seed.hardware))
		require.Empty(t, axesOf(t, s, seed.empty))
	})

	t.Run("la migración no cambia las variantes y restaura FORCE", func(t *testing.T) {
		s := dbtest.StartAt(t, attributesAdded)
		seed := seedFreeAxes(t, s)

		s.MigrateTo(t, axesByTenant)

		conn := connect(t, s.SuperuserURL)
		for id, want := range seed.axisValues {
			var raw []byte
			require.NoError(t, conn.QueryRow(t.Context(), "SELECT axis_values FROM product_variants WHERE id = $1", id).Scan(&raw))
			var got map[string]string
			require.NoError(t, json.Unmarshal(raw, &got))
			require.Equal(t, want, got)
		}
		for _, table := range []string{"variant_axes", "product_variants"} {
			var forced bool
			require.NoError(t, conn.QueryRow(t.Context(),
				"SELECT relforcerowsecurity FROM pg_class WHERE relname = $1", table).Scan(&forced))
			require.True(t, forced, "FORCE is restored on %s", table)
		}
	})

	t.Run("el Down quita los ejes", func(t *testing.T) {
		s := dbtest.StartAt(t, attributesAdded)
		seedFreeAxes(t, s)
		s.MigrateTo(t, axesByTenant)

		s.MigrateDownTo(t, attributesAdded)

		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT to_regclass('public.variant_axes') IS NOT NULL").Scan(&exists))
		require.False(t, exists)
	})

	t.Run("la migración falla si los conteos no cuadran", func(t *testing.T) {
		s := dbtest.StartAt(t, attributesAdded)
		seed := seedFreeAxes(t, s)
		// Simulates a backfill that silently stores fewer rows (as a FORCE
		// ROW LEVEL SECURITY left on would): when 00016 creates the table, an
		// event trigger (superuser only, test only) attaches a trigger that
		// drops the axes of one tenant.
		_, err := connect(t, s.SuperuserURL).Exec(t.Context(), `
			CREATE FUNCTION test_drop_one_tenant() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF NEW.tenant_id = '`+seed.hardware.String()+`' THEN RETURN NULL; END IF;
				RETURN NEW;
			END $$;
			CREATE FUNCTION test_attach_drop() RETURNS event_trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF EXISTS (SELECT 1 FROM pg_event_trigger_ddl_commands()
				           WHERE command_tag = 'CREATE TABLE' AND object_identity = 'public.variant_axes') THEN
					CREATE TRIGGER test_drop BEFORE INSERT ON public.variant_axes
						FOR EACH ROW EXECUTE FUNCTION test_drop_one_tenant();
				END IF;
			END $$;
			CREATE EVENT TRIGGER test_attach_drop ON ddl_command_end EXECUTE FUNCTION test_attach_drop();`)
		require.NoError(t, err)

		err = s.TryMigrateTo(t, axesByTenant)

		require.ErrorContains(t, err, "variant_axes backfill: the variants use 3 axes but 2 were stored")
		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT to_regclass('public.variant_axes') IS NOT NULL").Scan(&exists))
		require.False(t, exists, "the migration rolled back")
	})
}
