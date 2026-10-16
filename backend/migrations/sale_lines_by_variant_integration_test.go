//go:build integration

package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// Versions of the sale lines migration under test.
const (
	beforeSalesByVariant = 12 // 00012_stock_by_variant: sale lines still point to products only
	salesByVariant       = 13 // 00013_sale_lines_by_variant
)

// salesSeed is the data stored before 00013: products with their variants
// and the sales of each tenant, stored per product.
type salesSeed struct {
	tenantA, tenantB uuid.UUID
	// migrated is a product from before variants: its default variant has
	// its id. recent was created after variants: its default variant
	// (recentDefault) has its own id.
	migrated, recent, recentDefault uuid.UUID
	// other is a product of tenantB, from before variants.
	other uuid.UUID
	// sales are the sales stored, by tenant.
	sales map[uuid.UUID][]*domain.Sale
}

// seedSalesByProduct stores the seed as the schema owner, in transactions
// scoped to each tenant: a hardware store (tenantA) with a confirmed and a
// voided sale, and a clothing store (tenantB) with one sale. The lines are
// built by the sales domain, so their amounts add up, and stored in the
// columns that exist before 00013.
func seedSalesByProduct(t *testing.T, s *dbtest.Server) salesSeed {
	t.Helper()
	newID := func() uuid.UUID { return uuid.Must(uuid.NewV7()) }
	seed := salesSeed{
		tenantA: newID(), tenantB: newID(),
		migrated: newID(), recent: newID(), recentDefault: newID(), other: newID(),
		sales: map[uuid.UUID][]*domain.Sale{},
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
	product := func(tx pgx.Tx, tenant, id, variantID uuid.UUID, name, sku string) {
		exec(tx, `INSERT INTO products (id, tenant_id, name, description, base_unit, tax_rate, active, created_at)
			VALUES ($1, $2, $3, '', 'und', 0.19, true, $4)`, id, tenant, name, created)
		exec(tx, `INSERT INTO product_variants (id, tenant_id, product_id, sku, cost, price, is_default, active, created_at)
			VALUES ($1, $2, $3, $4, 1000, 2000, true, true, $5)`, variantID, tenant, id, sku, created)
	}
	location := func(tx pgx.Tx, tenant uuid.UUID) uuid.UUID {
		id := newID()
		exec(tx, "INSERT INTO locations (id, tenant_id, name, active, created_at) VALUES ($1, $2, 'Principal', true, $3)", id, tenant, created)
		return id
	}
	type line struct {
		product          uuid.UUID
		name             string
		qty, price, rate string
		discount         string
	}
	// sell stores a sale of lines, paid in cash with paid, as the sales
	// repository stored it before 00013; voided sales are voided after.
	number := int64(0)
	sell := func(tx pgx.Tx, tenant, locationID uuid.UUID, lines []line, paid string, voided bool) {
		sale, err := domain.NewSale(domain.NewSaleParams{
			ID: newID(), TenantID: tenant, LocationID: locationID, UserID: newID(), CreatedAt: created,
		})
		require.NoError(t, err)
		for _, l := range lines {
			require.NoError(t, sale.AddLine(domain.LineInput{
				ProductID: l.product, VariantID: l.product, ProductName: l.name, Quantity: dec(l.qty),
				UnitPrice: dec(l.price), TaxRate: dec(l.rate), Discount: dec(l.discount), PriceRule: domain.VariantPriceRule,
			}, domain.NoDiscountLimit))
		}
		cash, err := domain.NewPayment(domain.PaymentCash, dec(paid))
		require.NoError(t, err)
		require.NoError(t, sale.Confirm([]domain.Payment{cash}, created))
		number++
		require.NoError(t, sale.AssignNumber(number))
		if voided {
			require.NoError(t, sale.Void(newID(), "Error de digitación", created))
		}
		snap := sale.Snapshot()
		tot := snap.Totals
		exec(tx, `INSERT INTO sales (id, tenant_id, location_id, number, user_id, status, subtotal, discounts, tax, total,
				paid, change, created_at, confirmed_at)
			VALUES ($1, $2, $3, $4, $5, 'confirmed', $6, $7, $8, $9, $10, $11, $12, $12)`,
			snap.ID, tenant, locationID, snap.Number, snap.UserID, tot.Subtotal, tot.Discounts, tot.Tax, tot.Total,
			tot.Paid, tot.Change, created)
		for i, l := range snap.Lines {
			exec(tx, `INSERT INTO sale_lines (tenant_id, sale_id, position, product_id, product_name, quantity, unit_price,
					tax_rate, discount, gross, tax)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
				tenant, snap.ID, i, l.ProductID, l.ProductName, l.Quantity, l.UnitPrice, l.TaxRate, l.Discount, l.Gross, l.Tax)
		}
		exec(tx, "INSERT INTO sale_payments (tenant_id, sale_id, position, method, amount) VALUES ($1, $2, 0, 'cash', $3)",
			tenant, snap.ID, paid)
		if v := snap.Void; v != nil {
			exec(tx, "UPDATE sales SET status = 'voided', voided_at = $2, voided_by = $3, void_reason = $4 WHERE id = $1",
				snap.ID, v.At, v.UserID, v.Reason)
		}
		seed.sales[tenant] = append(seed.sales[tenant], sale)
	}

	for _, tenant := range []uuid.UUID{seed.tenantA, seed.tenantB} {
		inTenant(tenant, func(tx pgx.Tx) {
			exec(tx, "INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Negocio', '900123456-7', 'active', $2)", tenant, created)
		})
	}
	inTenant(seed.tenantA, func(tx pgx.Tx) {
		product(tx, seed.tenantA, seed.migrated, seed.migrated, "Cemento gris 50 kg", "CEM-50")
		product(tx, seed.tenantA, seed.recent, seed.recentDefault, "Cable 12 AWG", "CAB-12")
		loc := location(tx, seed.tenantA)
		sell(tx, seed.tenantA, loc, []line{
			{seed.migrated, "Cemento gris 50 kg", "2.5", "30000", "0.19", "1000"},
			{seed.recent, "Cable 12 AWG", "3.75", "2500.5", "0.19", "0"},
			{seed.migrated, "Cemento gris 50 kg", "1", "30000", "0.19", "0"},
		}, "150000", false)
		sell(tx, seed.tenantA, loc, []line{{seed.recent, "Cable 12 AWG", "10", "1234.5678", "0", "0"}}, "12345.68", true)
	})
	inTenant(seed.tenantB, func(tx pgx.Tx) {
		product(tx, seed.tenantB, seed.other, seed.other, "Camiseta básica", "CAM-BAS")
		number = 0
		sell(tx, seed.tenantB, location(tx, seed.tenantB), []line{
			{seed.other, "Camiseta básica", "2", "25210.084", "0.19", "210.08"},
		}, "60000", false)
	})
	return seed
}

// lineTotals are the count and the sums of the sale lines of a tenant.
type lineTotals struct {
	lines                          int
	quantity, gross, discount, tax decimal.Decimal
}

// linesPerTenant reads the totals of the lines of each tenant, as the
// superuser across tenants.
func linesPerTenant(t *testing.T, s *dbtest.Server) map[uuid.UUID]lineTotals {
	t.Helper()
	conn := connect(t, s.SuperuserURL)
	rows, err := conn.Query(t.Context(), `
		SELECT tenant_id, count(*), sum(quantity), sum(gross), sum(discount), sum(tax)
		FROM sale_lines GROUP BY tenant_id`)
	require.NoError(t, err)
	out := map[uuid.UUID]lineTotals{}
	for rows.Next() {
		var id uuid.UUID
		var lt lineTotals
		require.NoError(t, rows.Scan(&id, &lt.lines, &lt.quantity, &lt.gross, &lt.discount, &lt.tax))
		out[id] = lt
	}
	require.NoError(t, rows.Err())
	return out
}

func TestSaleLinesByVariantMigration(t *testing.T) {
	t.Run("las ventas existentes conservan exactamente sus totales y se rehidratan", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeSalesByVariant)
		seed := seedSalesByProduct(t, s)
		before := linesPerTenant(t, s)
		require.Len(t, before, 2)

		s.MigrateTo(t, salesByVariant)

		after := linesPerTenant(t, s)
		require.Len(t, after, 2)
		for tenant, want := range before {
			got := after[tenant]
			require.Equal(t, want.lines, got.lines, tenant)
			for _, pair := range [][2]decimal.Decimal{
				{want.quantity, got.quantity}, {want.gross, got.gross}, {want.discount, got.discount}, {want.tax, got.tax},
			} {
				require.True(t, pair[0].Equal(pair[1]), "tenant %s: %s before, %s after", tenant, pair[0], pair[1])
			}
		}

		// The repository rebuilds every sale with the amounts it had.
		repo := postgres.NewSaleRepository(dbtest.Open(t, s.AppURL))
		for tenant, sales := range seed.sales {
			for _, want := range sales {
				got, err := repo.Get(t.Context(), tenant, want.ID())
				require.NoError(t, err)
				require.Equal(t, want.Status(), got.Status())
				require.True(t, want.Totals().Total.Equal(got.Totals().Total), "sale %s", want.ID())
				require.True(t, want.Paid().Equal(got.Paid()))
				require.True(t, want.Change().Equal(got.Change()))
				require.Len(t, got.Lines(), len(want.Lines()))
				for i, l := range got.Lines() {
					require.Equal(t, want.Lines()[i].ProductID(), l.ProductID())
					require.True(t, want.Lines()[i].Total().Equal(l.Total()))
					require.Empty(t, l.VariantLabel())
					require.Equal(t, domain.VariantPriceRule, l.PriceRule())
				}
			}
		}
	})

	t.Run("cada línea queda en la variante por defecto de su producto", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeSalesByVariant)
		seed := seedSalesByProduct(t, s)

		s.MigrateTo(t, salesByVariant)

		conn := connect(t, s.SuperuserURL)
		rows, err := conn.Query(t.Context(), "SELECT product_id, variant_id, count(*) FROM sale_lines GROUP BY product_id, variant_id")
		require.NoError(t, err)
		got := map[[2]uuid.UUID]int{}
		for rows.Next() {
			var product, variant uuid.UUID
			var n int
			require.NoError(t, rows.Scan(&product, &variant, &n))
			got[[2]uuid.UUID{product, variant}] = n
		}
		require.NoError(t, rows.Err())
		require.Equal(t, map[[2]uuid.UUID]int{
			{seed.migrated, seed.migrated}:    2,
			{seed.recent, seed.recentDefault}: 2,
			{seed.other, seed.other}:          1,
		}, got)

		var forced bool
		require.NoError(t, conn.QueryRow(t.Context(),
			"SELECT bool_and(relforcerowsecurity) FROM pg_class WHERE relname IN ('sale_lines', 'product_variants')").Scan(&forced))
		require.True(t, forced, "FORCE is restored")
	})

	t.Run("el Down quita las columnas y deja las ventas como estaban", func(t *testing.T) {
		s := dbtest.StartAt(t, beforeSalesByVariant)
		seedSalesByProduct(t, s)
		before := linesPerTenant(t, s)

		s.MigrateTo(t, salesByVariant)
		s.MigrateDownTo(t, beforeSalesByVariant)

		require.Equal(t, len(before), len(linesPerTenant(t, s)))
		conn := connect(t, s.SuperuserURL)
		var columns int
		require.NoError(t, conn.QueryRow(t.Context(), `
			SELECT count(*) FROM information_schema.columns
			WHERE table_name = 'sale_lines' AND column_name IN ('variant_id', 'variant_label', 'price_rule')`).Scan(&columns))
		require.Zero(t, columns)
	})

	t.Run("el Down falla si hay líneas que el modelo anterior no puede representar", func(t *testing.T) {
		tests := []struct{ name, update string }{
			{"una línea con etiqueta de variante", "UPDATE sale_lines SET variant_label = 'Rojo' WHERE product_id = $1"},
			{"una línea con otra regla de precio", "UPDATE sale_lines SET price_rule = 'price_lists' WHERE product_id = $1"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := dbtest.StartAt(t, beforeSalesByVariant)
				seed := seedSalesByProduct(t, s)
				s.MigrateTo(t, salesByVariant)

				conn := connect(t, s.SuperuserURL)
				_, err := conn.Exec(t.Context(), tt.update, seed.recent)
				require.NoError(t, err)

				require.ErrorContains(t, s.TryMigrateDownTo(t, beforeSalesByVariant), "the previous model cannot represent")
			})
		}
	})
}
