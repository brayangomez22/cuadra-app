//go:build integration

package postgres_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 10, 10, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.LocationRepository    = (*postgres.LocationRepository)(nil)
	_ domain.StockRepository       = (*postgres.StockRepository)(nil)
	_ domain.PolicyRepository      = (*postgres.PolicyRepository)(nil)
	_ domain.ReorderRuleRepository = (*postgres.ReorderRepository)(nil)
	_ domain.StockCountRepository  = (*postgres.StockCountRepository)(nil)
	_ domain.CountEntryRepository  = (*postgres.CountEntryRepository)(nil)
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

type fixture struct {
	db        *db.DB
	locations *postgres.LocationRepository
	stock     *postgres.StockRepository
	policies  *postgres.PolicyRepository
	reorder   *postgres.ReorderRepository
	counts    *postgres.StockCountRepository
	entries   *postgres.CountEntryRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{
		db:        d,
		locations: postgres.NewLocationRepository(d),
		stock:     postgres.NewStockRepository(d),
		policies:  postgres.NewPolicyRepository(d),
		reorder:   postgres.NewReorderRepository(d),
		counts:    postgres.NewStockCountRepository(d),
		entries:   postgres.NewCountEntryRepository(d),
	}
}

// exec runs sql in a transaction scoped to tenantID.
func (f fixture) exec(t *testing.T, tenantID uuid.UUID, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), sql, args...)
		return err
	}))
}

// createTenant stores a tenant row directly: inventory does not depend on
// the identity module, but its tables reference tenants.
func (f fixture) createTenant(t *testing.T) uuid.UUID {
	t.Helper()
	id := newID()
	f.exec(t, id, "INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Ferretería', '900123456-8', 'active', 'ferreteria', $2)", id, now)
	return id
}

// createProduct stores a product row and its variants directly, for the same
// reason: the default variant and extra more. Like a product created after
// variants existed, no variant has the product's id. It returns the
// variant ids, the default one first.
func (f fixture) createProduct(t *testing.T, tenantID uuid.UUID, extra int) []uuid.UUID {
	t.Helper()
	productID := newID()
	f.exec(t, tenantID, `INSERT INTO units_of_measure (id, tenant_id, code, name, created_at)
		VALUES ($1, $2, 'und', 'Unidad', $3) ON CONFLICT (tenant_id, code) DO NOTHING`, newID(), tenantID, now)
	f.exec(t, tenantID, `INSERT INTO products (id, tenant_id, name, description, base_unit_id, tax_rate, active, created_at)
		VALUES ($1, $2, 'Producto', '', (SELECT id FROM units_of_measure WHERE code = 'und'), 0.19, true, $3)`,
		productID, tenantID, now)
	ids := make([]uuid.UUID, 0, extra+1)
	for i := range extra + 1 {
		id := newID()
		f.exec(t, tenantID, `INSERT INTO product_variants (id, tenant_id, product_id, sku, cost, price, axis_values, is_default, active, created_at)
			VALUES ($1, $2, $3, $4, 30000, 35000, jsonb_build_object('opcion', $5::text), $6, true, $7)`,
			id, tenantID, productID, "SKU-"+strings.ToUpper(id.String()), strconv.Itoa(i), i == 0, now)
		ids = append(ids, id)
	}
	return ids
}

// createVariant stores a simple product and returns its default variant.
func (f fixture) createVariant(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return f.createProduct(t, tenantID, 0)[0]
}

// levelRows counts the stored levels of a variant at a location, lot or not.
func (f fixture) levelRows(t *testing.T, tenantID, variantID, locationID uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "SELECT count(*) FROM stock_levels WHERE variant_id = $1 AND location_id = $2",
			variantID, locationID).Scan(&n)
	}))
	return n
}

func (f fixture) createLocation(t *testing.T, tenantID uuid.UUID, name string) *domain.Location {
	t.Helper()
	l, err := domain.NewLocation(tenantID, name, now)
	require.NoError(t, err)
	require.NoError(t, f.locations.Create(t.Context(), l))
	return l
}

func info(userID uuid.UUID) domain.MovementInfo {
	return domain.MovementInfo{UserID: userID, OccurredAt: now}
}

// receive registers an entry through Apply, as the use case does.
func (f fixture) receive(t *testing.T, tenantID, variantID, locationID uuid.UUID, qty, cost string) {
	t.Helper()
	err := f.stock.Apply(t.Context(), tenantID, variantID, []uuid.UUID{locationID},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			m, err := levels[0].Receive(dec(qty), dec(cost), info(newID()))
			return []*domain.StockMovement{m}, err
		})
	require.NoError(t, err)
}

// level returns the stored level of a variant at a location; zero if none.
func (f fixture) level(t *testing.T, tenantID, variantID, locationID uuid.UUID) decimal.Decimal {
	t.Helper()
	page, err := f.stock.ListLevels(t.Context(), tenantID, domain.StockFilter{LocationID: locationID, VariantID: &variantID, Limit: 10})
	require.NoError(t, err)
	if page.Total == 0 {
		return decimal.Zero
	}
	return page.Items[0].Quantity()
}
