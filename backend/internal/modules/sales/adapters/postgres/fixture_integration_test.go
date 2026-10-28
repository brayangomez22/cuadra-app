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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 10, 10, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.SaleRepository   = (*postgres.SaleRepository)(nil)
	_ domain.PolicyRepository = (*postgres.PolicyRepository)(nil)
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func noReserve(context.Context) error { return nil }

type fixture struct {
	db       *db.DB
	sales    *postgres.SaleRepository
	policies *postgres.PolicyRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{db: d, sales: postgres.NewSaleRepository(d), policies: postgres.NewPolicyRepository(d)}
}

func (f fixture) exec(t *testing.T, tenantID uuid.UUID, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), sql, args...)
		return err
	}))
}

// tenant is a tenant with a location and a product, stored directly: sales
// does not depend on the other modules, but its tables reference theirs.
// The product's default variant has its id, as a product from before
// variants; variant is a second variant with its own id.
type tenant struct {
	id, location, product, variant uuid.UUID
	// customer is a customer of the tenant: sales reference customers with
	// a foreign key.
	customer uuid.UUID
}

func (f fixture) createTenant(t *testing.T) tenant {
	t.Helper()
	tn := tenant{id: newID(), product: newID(), variant: newID(), customer: newID()}
	f.exec(t, tn.id, "INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Ferretería', '900123456-8', 'active', 'ferreteria', $2)", tn.id, now)
	f.exec(t, tn.id, "INSERT INTO units_of_measure (id, tenant_id, code, name, created_at) VALUES ($1, $2, 'und', 'Unidad', $3)",
		newID(), tn.id, now)
	f.exec(t, tn.id, `INSERT INTO products (id, tenant_id, name, description, base_unit_id, tax_rate, active, created_at)
		VALUES ($1, $2, 'Cemento gris 50kg', '', (SELECT id FROM units_of_measure WHERE code = 'und'), 0.19, true, $3)`,
		tn.product, tn.id, now)
	f.exec(t, tn.id, `INSERT INTO product_variants (id, tenant_id, product_id, sku, cost, price, is_default, active, created_at)
		VALUES ($1, $2, $1, $3, 25000, 30000, true, true, $4)`,
		tn.product, tn.id, "SKU-"+strings.ToUpper(tn.product.String()), now)
	f.exec(t, tn.id, `INSERT INTO product_variants (id, tenant_id, product_id, sku, cost, price, axis_values, is_default, active, created_at)
		VALUES ($1, $2, $3, $4, 12000, 15000, '{"presentación": "25kg"}', false, true, $5)`,
		tn.variant, tn.id, tn.product, "SKU-"+strings.ToUpper(tn.variant.String()), now)
	f.exec(t, tn.id, `INSERT INTO customers (id, tenant_id, person_type, document_type, document_number, name,
		tax_responsibilities, active, created_at, updated_at)
		VALUES ($1, $2, 'natural', 'cc', '1020304050', 'Cliente', '{R-99-PN}', true, $3, $3)`, tn.customer, tn.id, now)
	tn.location = f.createLocation(t, tn.id)
	return tn
}

func (f fixture) createLocation(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := newID()
	f.exec(t, tenantID, "INSERT INTO locations (id, tenant_id, name, active, created_at) VALUES ($1, $2, $3, true, $4)",
		id, tenantID, "Sede "+id.String(), now)
	return id
}

// confirmed builds a confirmed sale of the tenant's product at location, in
// two lines at 19%: 2.5 × 30000 of the default variant with a discount of
// 1000 (base 74000, IVA 14060) and 1 × 30000 of the second variant, with its
// label and a price list's price (IVA 5700); total 123760.00, paid with
// 50000 by Nequi and 80000 in cash.
func confirmed(t *testing.T, tn tenant, location uuid.UUID, confirmedAt time.Time) *domain.Sale {
	t.Helper()
	customer := tn.customer
	return confirmedWith(t, tn, domain.NewSaleParams{
		ID: newID(), TenantID: tn.id, LocationID: location, UserID: newID(), CustomerID: &customer, CreatedAt: confirmedAt,
	})
}

// confirmedWith is confirmed with the given sale fields.
func confirmedWith(t *testing.T, tn tenant, p domain.NewSaleParams) *domain.Sale {
	t.Helper()
	confirmedAt := p.CreatedAt
	s, err := domain.NewSale(p)
	require.NoError(t, err)
	require.NoError(t, s.AddLine(domain.LineInput{
		ProductID: tn.product, VariantID: tn.product, ProductName: "Cemento gris 50kg", Quantity: dec("2.5"),
		UnitPrice: dec("30000"), TaxRate: dec("0.19"), Discount: dec("1000"), PriceRule: domain.VariantPriceRule,
	}, domain.NoDiscountLimit))
	require.NoError(t, s.AddLine(domain.LineInput{
		ProductID: tn.product, VariantID: tn.variant, ProductName: "Cemento gris 50kg", VariantLabel: "25kg",
		Quantity: dec("1"), UnitPrice: dec("30000"), TaxRate: dec("0.19"), Discount: dec("0"), PriceRule: "price_lists",
	}, domain.NoDiscountLimit))
	nequi, err := domain.NewPayment(domain.PaymentNequi, dec("50000"))
	require.NoError(t, err)
	cash, err := domain.NewPayment(domain.PaymentCash, dec("80000"))
	require.NoError(t, err)
	require.NoError(t, s.Confirm([]domain.Payment{nequi, cash}, confirmedAt))
	return s
}

// norm drops the trailing zeros that NUMERIC(18,4) adds (2.5000 → 2.5), so
// decimals of equal value compare equal.
func norm(d decimal.Decimal) decimal.Decimal { return decimal.RequireFromString(d.String()) }

// normalized returns snap with every decimal normalized.
func normalized(snap domain.SaleSnapshot) domain.SaleSnapshot {
	for i, l := range snap.Lines {
		snap.Lines[i] = domain.LineSnapshot{
			ProductID: l.ProductID, VariantID: l.VariantID, ProductName: l.ProductName, VariantLabel: l.VariantLabel,
			Quantity: norm(l.Quantity), UnitPrice: norm(l.UnitPrice), TaxRate: norm(l.TaxRate), Discount: norm(l.Discount),
			Gross: norm(l.Gross), Tax: norm(l.Tax), PriceRule: l.PriceRule,
		}
	}
	for i, p := range snap.Payments {
		snap.Payments[i].Amount = norm(p.Amount)
	}
	t := snap.Totals
	snap.Totals = domain.StoredTotals{
		Subtotal: norm(t.Subtotal), Discounts: norm(t.Discounts), Tax: norm(t.Tax),
		Total: norm(t.Total), Paid: norm(t.Paid), Change: norm(t.Change),
	}
	return snap
}

func (f fixture) save(t *testing.T, s *domain.Sale) {
	t.Helper()
	require.NoError(t, f.sales.SaveConfirmed(t.Context(), s, noReserve))
}
