//go:build integration

package migrations_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// Versions of the migrations under test.
const (
	beforeVariants   = 9  // 00009_cash_sessions: products still hold SKU, barcode, cost and price
	variantsCreated  = 10 // 00010_product_variants
	variantColsMoved = 11 // 00011_products_drop_variant_columns
)

var created = time.Date(2026, 10, 1, 14, 0, 0, 0, time.UTC)

// legacyProduct is a row of products before variants existed.
type legacyProduct struct {
	id, tenantID uuid.UUID
	sku          string
	barcode      *string
	cost, price  decimal.Decimal
	active       bool
}

func ptr(s string) *string { return &s }

func connect(t *testing.T, url string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(t.Context())) })
	return conn
}

// seedLegacy stores two tenants with products of the model before variants,
// as the schema owner: inside a transaction scoped to each tenant, the way
// RLS lets the owner write.
func seedLegacy(t *testing.T, s *dbtest.Server) []legacyProduct {
	t.Helper()
	tenantA, tenantB := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	products := []legacyProduct{
		{uuid.Must(uuid.NewV7()), tenantA, "CEM-GRIS-50", ptr("7701234567890"), dec("28000.1234"), dec("32500.5"), true},
		{uuid.Must(uuid.NewV7()), tenantA, "TUB-PVC-12", nil, dec("0"), dec("4500"), false},
		{uuid.Must(uuid.NewV7()), tenantB, "CAM-BASICA", ptr("7709876543210"), dec("15000"), dec("29900"), true},
	}
	conn := connect(t, s.AdminURL)
	for _, tenant := range []uuid.UUID{tenantA, tenantB} {
		tx, err := conn.Begin(t.Context())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(), "SELECT set_config('app.tenant_id', $1, true)", tenant.String())
		require.NoError(t, err)
		_, err = tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Negocio', '900123456-7', 'active', $2)", tenant, created)
		require.NoError(t, err)
		for _, p := range products {
			if p.tenantID != tenant {
				continue
			}
			_, err = tx.Exec(t.Context(), `
				INSERT INTO products (id, tenant_id, sku, barcode, name, description, base_unit, cost, price, tax_rate, active, created_at)
				VALUES ($1, $2, $3, $4, 'Producto '||$3, '', 'und', $5, $6, 0.19, $7, $8)`,
				p.id, p.tenantID, p.sku, p.barcode, p.cost, p.price, p.active, created)
			require.NoError(t, err)
		}
		require.NoError(t, tx.Commit(t.Context()))
	}
	return products
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestProductVariantsMigration(t *testing.T) {
	t.Run("las migraciones corren con un dueño que no es superusuario", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeVariants)
		var super, bypass bool
		require.NoError(t, connect(t, s.AdminURL).QueryRow(t.Context(),
			"SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user").Scan(&super, &bypass))
		require.False(t, super)
		require.False(t, bypass)
	})

	t.Run("conserva SKU, código, costo y precio de cada producto y la variante por defecto tiene su mismo ID", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeVariants)
		products := seedLegacy(t, s)

		s.MigrateTo(t, variantColsMoved)

		// The superuser reads every tenant without RLS.
		conn := connect(t, s.SuperuserURL)
		var count int
		require.NoError(t, conn.QueryRow(t.Context(), "SELECT count(*) FROM product_variants").Scan(&count))
		require.Equal(t, len(products), count)
		for _, p := range products {
			var (
				productID, tenantID uuid.UUID
				sku                 string
				barcode             *string
				cost, price         decimal.Decimal
				axes                map[string]string
				isDefault, active   bool
				createdAt           time.Time
			)
			require.NoError(t, conn.QueryRow(t.Context(), `
				SELECT product_id, tenant_id, sku, barcode, cost, price, axis_values, is_default, active, created_at
				FROM product_variants WHERE id = $1`, p.id,
			).Scan(&productID, &tenantID, &sku, &barcode, &cost, &price, &axes, &isDefault, &active, &createdAt), p.sku)

			require.Equal(t, p.id, productID, p.sku)
			require.Equal(t, p.tenantID, tenantID, p.sku)
			require.Equal(t, p.sku, sku)
			require.Equal(t, p.barcode, barcode, p.sku)
			require.True(t, p.cost.Equal(cost), p.sku)
			require.True(t, p.price.Equal(price), p.sku)
			require.Empty(t, axes, p.sku)
			require.True(t, isDefault, p.sku)
			require.True(t, active, "the product keeps its own status: %s", p.sku)
			require.Equal(t, created, createdAt.UTC(), p.sku)
		}

		var moved int
		require.NoError(t, conn.QueryRow(t.Context(), `
			SELECT count(*) FROM information_schema.columns
			WHERE table_name = 'products' AND column_name IN ('sku', 'barcode', 'cost', 'price')`).Scan(&moved))
		require.Zero(t, moved, "the variant columns left products")

		var forced bool
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT relforcerowsecurity FROM pg_class WHERE relname = 'product_variants'").Scan(&forced))
		require.True(t, forced)
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT relforcerowsecurity FROM pg_class WHERE relname = 'products'").Scan(&forced))
		require.True(t, forced, "FORCE is restored after the backfill")
	})

	t.Run("el Down de 00011 restaura las columnas desde la variante por defecto", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeVariants)
		products := seedLegacy(t, s)
		s.MigrateTo(t, variantColsMoved)

		s.MigrateDownTo(t, variantsCreated)

		conn := connect(t, s.SuperuserURL)
		for _, p := range products {
			var sku string
			var barcode *string
			var cost, price decimal.Decimal
			require.NoError(t, conn.QueryRow(t.Context(),
				"SELECT sku, barcode, cost, price FROM products WHERE id = $1", p.id).Scan(&sku, &barcode, &cost, &price))
			require.Equal(t, p.sku, sku)
			require.Equal(t, p.barcode, barcode, p.sku)
			require.True(t, p.cost.Equal(cost), p.sku)
			require.True(t, p.price.Equal(price), p.sku)
		}
		var forced bool
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT relforcerowsecurity FROM pg_class WHERE relname = 'products'").Scan(&forced))
		require.True(t, forced)
	})

	t.Run("la migración falla si los conteos no cuadran", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeVariants)
		products := seedLegacy(t, s)
		// Simulates a backfill that silently copies fewer rows (as a FORCE
		// ROW LEVEL SECURITY left on would): when 00010 creates the table, an
		// event trigger (superuser only, test only) attaches a trigger that
		// drops the rows of one tenant.
		_, err := connect(t, s.SuperuserURL).Exec(t.Context(), `
			CREATE FUNCTION test_drop_one_tenant() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF NEW.tenant_id = '`+products[2].tenantID.String()+`' THEN RETURN NULL; END IF;
				RETURN NEW;
			END $$;
			CREATE FUNCTION test_attach_drop() RETURNS event_trigger LANGUAGE plpgsql AS $$
			BEGIN
				IF EXISTS (SELECT 1 FROM pg_event_trigger_ddl_commands()
				           WHERE command_tag = 'CREATE TABLE' AND object_identity = 'public.product_variants') THEN
					CREATE TRIGGER test_drop BEFORE INSERT ON public.product_variants
						FOR EACH ROW EXECUTE FUNCTION test_drop_one_tenant();
				END IF;
			END $$;
			CREATE EVENT TRIGGER test_attach_drop ON ddl_command_end EXECUTE FUNCTION test_attach_drop();`)
		require.NoError(t, err)

		err = s.TryMigrateTo(t, variantsCreated)

		require.ErrorContains(t, err, "product_variants backfill: 3 products but 2 matching default variants")
		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(),
			"SELECT to_regclass('public.product_variants') IS NOT NULL").Scan(&exists))
		require.False(t, exists, "the migration rolled back")
	})
}
