//go:build integration

package postgres_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through
// timestamptz.
var now = time.Date(2026, 10, 28, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.SupplierRepository = (*postgres.SupplierRepository)(nil)
	_ domain.PurchaseRepository = (*postgres.PurchaseRepository)(nil)
)

type fixture struct {
	db        *db.DB
	suppliers *postgres.SupplierRepository
	purchases *postgres.PurchaseRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{db: d, suppliers: postgres.NewSupplierRepository(d), purchases: postgres.NewPurchaseRepository(d)}
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// tenant is a tenant with a location and one variant with an alternate unit
// (a box of 100).
type tenant struct {
	id, location, variant, box uuid.UUID
}

// createTenant stores a tenant, a location and a product directly:
// purchases depends on neither identity, inventory nor catalog, but its
// tables reference them.
func (f fixture) createTenant(t *testing.T) tenant {
	t.Helper()
	tn := tenant{id: newID(), location: newID(), variant: newID(), box: newID()}
	product := newID()
	f.exec(t, tn.id, "INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Negocio', '900123456-8', 'active', 'generica', $2)", tn.id, now)
	f.exec(t, tn.id, "INSERT INTO locations (id, tenant_id, name, active, created_at) VALUES ($1, $2, 'Bodega', true, $3)", tn.location, tn.id, now)
	f.exec(t, tn.id, `INSERT INTO units_of_measure (id, tenant_id, code, name, created_at)
		VALUES ($1, $2, 'und', 'Unidad', $4), ($3, $2, 'caja', 'Caja', $4)`, newID(), tn.id, tn.box, now)
	f.exec(t, tn.id, `INSERT INTO products (id, tenant_id, name, description, base_unit_id, tax_rate, active, created_at)
		VALUES ($1, $2, 'Producto', '', (SELECT id FROM units_of_measure WHERE code = 'und'), 0.19, true, $3)`, product, tn.id, now)
	f.exec(t, tn.id, `INSERT INTO product_variants (id, tenant_id, product_id, sku, cost, price, axis_values, is_default, active, created_at)
		VALUES ($1, $2, $3, $4, 0, 1000, '{}', true, true, $5)`, tn.variant, tn.id, product, "SKU-"+strings.ToUpper(tn.variant.String()), now)
	return tn
}

func (f fixture) exec(t *testing.T, tenantID uuid.UUID, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), sql, args...)
		return err
	}))
}

// addSupplier stores an active supplier with a 30-day term.
func (f fixture) addSupplier(t *testing.T, tn tenant, name, nit string) *domain.Supplier {
	t.Helper()
	s, err := domain.NewSupplier(tn.id, domain.SupplierParams{
		Name: name, NIT: nit, PaymentTermDays: 30, Active: true,
		Contact: domain.Contact{Name: "Laura Pérez", Phone: "604 444 5566"},
	}, now)
	require.NoError(t, err)
	require.NoError(t, f.suppliers.Create(t.Context(), s))
	return s
}

// newPurchase is a purchase of 2 boxes of the tenant's variant, received as
// 200 base units.
func newPurchase(t *testing.T, tn tenant, s *domain.Supplier, invoice string, pay domain.Payment) *domain.Purchase {
	t.Helper()
	p, err := domain.NewPurchase(tn.id, domain.PurchaseParams{
		Supplier: s, LocationID: tn.location, InvoiceNumber: invoice, IssuedOn: domain.NewDate(2026, time.October, 20),
		Payment: pay, Notes: "Llegó completa", CreatedBy: newID(),
	}, []domain.LineInput{{
		VariantID: tn.variant, ProductName: "Producto", Quantity: dec("2"), UnitCost: dec("45000"), TaxRate: dec("0.19"),
		Unit: domain.LineUnit{ID: &tn.box, Code: "caja", Factor: dec("100")},
	}}, now)
	require.NoError(t, err)
	return p
}

// receive is what the inventory records for newPurchase's line.
func receive(p *domain.Purchase) func(context.Context) error {
	return func(context.Context) error {
		return p.Receive([]domain.Received{{BaseQuantity: dec("200"), BaseUnitCost: dec("450")}})
	}
}

// save stores p, received as newPurchase's line.
func (f fixture) save(t *testing.T, p *domain.Purchase) {
	t.Helper()
	require.NoError(t, f.purchases.Save(t.Context(), p, receive(p)))
}
