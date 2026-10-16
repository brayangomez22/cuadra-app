//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// Versions of the stock migration under test.
const (
	beforeStockByVariant = 11 // 00011_products_drop_variant_columns: stock still points to products
	stockByVariant       = 12 // 00012_stock_by_variant
)

// stockSeed is the data stored before 00012: products, their variants and
// the stock and kardex recorded per product.
type stockSeed struct {
	tenantA, tenantB uuid.UUID
	// migrated is a product from before variants: its default variant has
	// its id. recent was created after variants: its default variant has
	// its own id, and it has a second variant without stock.
	migrated, recent, recentDefault, recentOther uuid.UUID
	// other is a product of tenantB, from before variants.
	other uuid.UUID
}

// seedStockByProduct stores the seed as the schema owner, in transactions
// scoped to each tenant: a hardware store (tenantA) and a clothing store
// (tenantB).
func seedStockByProduct(t *testing.T, s *dbtest.Server) stockSeed {
	t.Helper()
	newID := func() uuid.UUID { return uuid.Must(uuid.NewV7()) }
	seed := stockSeed{
		tenantA: newID(), tenantB: newID(),
		migrated: newID(), recent: newID(), recentDefault: newID(), recentOther: newID(),
		other: newID(),
	}
	conn := connect(t, s.AdminURL)
	inTenant := func(tenant uuid.UUID, fn func(tx pgx.Tx)) {
		tx, err := conn.Begin(t.Context())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(), "SELECT set_config('app.tenant_id', $1, true)", tenant.String())
		require.NoError(t, err)
		fn(tx)
		require.NoError(t, tx.Commit(t.Context()))
	}
	exec := func(tx pgx.Tx, sql string, args ...any) {
		_, err := tx.Exec(t.Context(), sql, args...)
		require.NoError(t, err, sql)
	}
	product := func(tx pgx.Tx, tenant, id uuid.UUID, name string) {
		exec(tx, `INSERT INTO products (id, tenant_id, name, description, base_unit, tax_rate, active, created_at)
			VALUES ($1, $2, $3, '', 'und', 0.19, true, $4)`, id, tenant, name, created)
	}
	variant := func(tx pgx.Tx, tenant, id, productID uuid.UUID, sku string, axes string, isDefault bool) {
		exec(tx, `INSERT INTO product_variants (id, tenant_id, product_id, sku, cost, price, axis_values, is_default, active, created_at)
			VALUES ($1, $2, $3, $4, 1000, 2000, $5::jsonb, $6, true, $7)`, id, tenant, productID, sku, axes, isDefault, created)
	}
	location := func(tx pgx.Tx, tenant uuid.UUID, name string) uuid.UUID {
		id := newID()
		exec(tx, "INSERT INTO locations (id, tenant_id, name, active, created_at) VALUES ($1, $2, $3, true, $4)", id, tenant, name, created)
		return id
	}
	// receive stores a level and the purchase_in that explains it, by product.
	receive := func(tx pgx.Tx, tenant, productID, locationID uuid.UUID, qty string) {
		exec(tx, `INSERT INTO stock_levels (tenant_id, product_id, location_id, quantity, average_cost, updated_at)
			VALUES ($1, $2, $3, $4, 1000, $5)`, tenant, productID, locationID, qty, created)
		exec(tx, `INSERT INTO stock_movements (id, tenant_id, product_id, location_id, type, quantity, unit_cost, balance_after,
				average_cost_after, reason, user_id, occurred_at)
			VALUES ($1, $2, $3, $4, 'purchase_in', $5, 1000, $5, 1000, '', $6, $7)`, newID(), tenant, productID, locationID, qty, newID(), created)
	}

	for _, tenant := range []uuid.UUID{seed.tenantA, seed.tenantB} {
		inTenant(tenant, func(tx pgx.Tx) {
			exec(tx, "INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Negocio', '900123456-7', 'active', $2)", tenant, created)
		})
	}
	inTenant(seed.tenantA, func(tx pgx.Tx) {
		product(tx, seed.tenantA, seed.migrated, "Cemento gris 50 kg")
		variant(tx, seed.tenantA, seed.migrated, seed.migrated, "CEM-50", "{}", true)
		product(tx, seed.tenantA, seed.recent, "Cable 12 AWG")
		variant(tx, seed.tenantA, seed.recentDefault, seed.recent, "CAB-12-R", `{"color": "Rojo"}`, true)
		variant(tx, seed.tenantA, seed.recentOther, seed.recent, "CAB-12-N", `{"color": "Negro"}`, false)
		bodega, centro := location(tx, seed.tenantA, "Bodega"), location(tx, seed.tenantA, "Centro")
		receive(tx, seed.tenantA, seed.migrated, bodega, "10")
		receive(tx, seed.tenantA, seed.migrated, centro, "3.5")
		receive(tx, seed.tenantA, seed.recent, bodega, "7.25")
	})
	inTenant(seed.tenantB, func(tx pgx.Tx) {
		product(tx, seed.tenantB, seed.other, "Camiseta básica")
		variant(tx, seed.tenantB, seed.other, seed.other, "CAM-BAS", "{}", true)
		receive(tx, seed.tenantB, seed.other, location(tx, seed.tenantB, "Almacén"), "4")
	})
	return seed
}

// stockPerProduct is the stock of each product (summed over its levels) and
// the count and sum of its movements, read by the superuser across tenants.
type productStock struct {
	quantity, moved decimal.Decimal
	levels, movs    int
}

func stockPerProduct(t *testing.T, s *dbtest.Server, levelsSQL, movementsSQL string) map[uuid.UUID]productStock {
	t.Helper()
	conn := connect(t, s.SuperuserURL)
	out := map[uuid.UUID]productStock{}
	rows, err := conn.Query(t.Context(), levelsSQL)
	require.NoError(t, err)
	for rows.Next() {
		var id uuid.UUID
		var ps productStock
		require.NoError(t, rows.Scan(&id, &ps.quantity, &ps.levels))
		out[id] = ps
	}
	require.NoError(t, rows.Err())
	rows, err = conn.Query(t.Context(), movementsSQL)
	require.NoError(t, err)
	for rows.Next() {
		var id uuid.UUID
		var moved decimal.Decimal
		var movs int
		require.NoError(t, rows.Scan(&id, &moved, &movs))
		ps := out[id]
		ps.moved, ps.movs = moved, movs
		out[id] = ps
	}
	require.NoError(t, rows.Err())
	return out
}

func TestStockByVariantMigration(t *testing.T) {
	t.Run("el stock por producto antes de la migración es igual a la suma por variante después", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeStockByVariant)
		seed := seedStockByProduct(t, s)
		before := stockPerProduct(t, s,
			"SELECT product_id, sum(quantity), count(*) FROM stock_levels GROUP BY product_id",
			"SELECT product_id, sum(quantity), count(*) FROM stock_movements GROUP BY product_id")

		s.MigrateTo(t, stockByVariant)

		after := stockPerProduct(t, s, `
			SELECT v.product_id, sum(l.quantity), count(*)
			FROM stock_levels l JOIN product_variants v ON v.tenant_id = l.tenant_id AND v.id = l.variant_id
			GROUP BY v.product_id`, `
			SELECT v.product_id, sum(m.quantity), count(*)
			FROM stock_movements m JOIN product_variants v ON v.tenant_id = m.tenant_id AND v.id = m.variant_id
			GROUP BY v.product_id`)
		require.Len(t, after, 3)
		for id, want := range before {
			got := after[id]
			require.True(t, want.quantity.Equal(got.quantity), "product %s: stock %s before, %s after", id, want.quantity, got.quantity)
			require.True(t, want.moved.Equal(got.moved), "product %s: kardex %s before, %s after", id, want.moved, got.moved)
			require.Equal(t, want.levels, got.levels, id)
			require.Equal(t, want.movs, got.movs, id)
		}

		// Each product's stock is now on its default variant: the migrated
		// one keeps its id, the recent one moves to its own variant id.
		conn := connect(t, s.SuperuserURL)
		var onVariant map[uuid.UUID]int
		rows, err := conn.Query(t.Context(), `
			SELECT variant_id, count(*) FROM (
				SELECT variant_id FROM stock_levels UNION ALL SELECT variant_id FROM stock_movements
			) rows GROUP BY variant_id`)
		require.NoError(t, err)
		onVariant = map[uuid.UUID]int{}
		for rows.Next() {
			var id uuid.UUID
			var n int
			require.NoError(t, rows.Scan(&id, &n))
			onVariant[id] = n
		}
		require.NoError(t, rows.Err())
		require.Equal(t, map[uuid.UUID]int{seed.migrated: 4, seed.recentDefault: 2, seed.other: 2}, onVariant)

		var withLot int
		require.NoError(t, conn.QueryRow(t.Context(), `
			SELECT (SELECT count(*) FROM stock_levels WHERE lot_id IS NOT NULL)
			     + (SELECT count(*) FROM stock_movements WHERE lot_id IS NOT NULL)`).Scan(&withLot))
		require.Zero(t, withLot)

		for _, table := range []string{"stock_levels", "stock_movements", "product_variants"} {
			var forced bool
			require.NoError(t, conn.QueryRow(t.Context(),
				"SELECT relforcerowsecurity FROM pg_class WHERE relname = $1", table).Scan(&forced))
			require.True(t, forced, "FORCE is restored on %s", table)
		}
	})

	t.Run("el Down devuelve el stock a cada producto", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeStockByVariant)
		seed := seedStockByProduct(t, s)
		s.MigrateTo(t, stockByVariant)

		s.MigrateDownTo(t, beforeStockByVariant)

		var onRecent int
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT count(*) FROM stock_levels WHERE product_id = $1", seed.recent).Scan(&onRecent))
		require.Equal(t, 1, onRecent)
	})

	t.Run("la migración falla si los conteos no cuadran", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeStockByVariant)
		seedStockByProduct(t, s)
		// Simulates a backfill that changes the stock while moving it to the
		// variants: a trigger (test only) adds one unit to every level the
		// migration rewrites.
		_, err := connect(t, s.SuperuserURL).Exec(t.Context(), `
			CREATE FUNCTION test_skew_stock() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				NEW.quantity = NEW.quantity + 1;
				RETURN NEW;
			END $$;
			CREATE TRIGGER test_skew BEFORE UPDATE ON stock_levels FOR EACH ROW EXECUTE FUNCTION test_skew_stock();`)
		require.NoError(t, err)

		err = s.TryMigrateTo(t, stockByVariant)

		require.ErrorContains(t, err, "stock by variant: 1 products whose stock changed")
		var stillByProduct bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(), `
			SELECT EXISTS (SELECT 1 FROM information_schema.columns
			               WHERE table_name = 'stock_levels' AND column_name = 'product_id')`).Scan(&stillByProduct))
		require.True(t, stillByProduct, "the migration rolled back")
	})
}
