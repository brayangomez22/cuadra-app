//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 10, 10, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.ProductRepository  = (*postgres.ProductRepository)(nil)
	_ domain.CategoryRepository = (*postgres.CategoryRepository)(nil)
	_ domain.UnitRepository     = (*postgres.UnitRepository)(nil)
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

type fixture struct {
	db         *db.DB
	products   *postgres.ProductRepository
	categories *postgres.CategoryRepository
	units      *postgres.UnitRepository
	attributes *postgres.AttributeRepository
	axes       *postgres.VariantAxisRepository
	// piece is the "und" unit createTenant gives each tenant.
	piece map[uuid.UUID]domain.UnitRef
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{
		db: d, products: postgres.NewProductRepository(d), categories: postgres.NewCategoryRepository(d),
		units: postgres.NewUnitRepository(d), attributes: postgres.NewAttributeRepository(d),
		axes: postgres.NewVariantAxisRepository(d), piece: map[uuid.UUID]domain.UnitRef{},
	}
}

// createTenant stores a tenant row directly (the catalog does not depend on
// the identity module, but its tables reference tenants) and gives it the
// unit "und".
func (f fixture) createTenant(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	require.NoError(t, f.db.WithTenantTx(t.Context(), id, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Ferretería', '900123456-8', 'active', 'ferreteria', $2)", id, now)
		return err
	}))
	f.piece[id] = f.createUnit(t, id, "und", "Unidad").Ref()
	return id
}

func (f fixture) createUnit(t *testing.T, tenantID uuid.UUID, code, name string) *domain.UnitOfMeasure {
	t.Helper()
	u, err := domain.NewUnit(tenantID, code, name, now)
	require.NoError(t, err)
	require.NoError(t, f.units.Create(t.Context(), u))
	return u
}

type productOption func(*domain.NewProductParams)

func withBarcode(code string) productOption {
	return func(p *domain.NewProductParams) { p.Variant.Barcode = code }
}

func withAttributes(attrs domain.Attributes) productOption {
	return func(p *domain.NewProductParams) { p.Attributes = attrs }
}

func withCategory(id uuid.UUID) productOption {
	return func(p *domain.NewProductParams) { p.CategoryID = &id }
}

func (f fixture) newProduct(t *testing.T, tenantID uuid.UUID, sku, name string, opts ...productOption) *domain.Product {
	t.Helper()
	rate, err := domain.ParseTaxRate(dec("0.19"))
	require.NoError(t, err)
	params := domain.NewProductParams{
		TenantID: tenantID, Name: name, Description: "Producto de prueba", BaseUnit: f.piece[tenantID], TaxRate: rate,
		Variant: domain.VariantParams{SKU: sku, Cost: dec("8333.3333"), Price: dec("10000.1234")},
	}
	for _, opt := range opts {
		opt(&params)
	}
	p, err := domain.NewProduct(params, now)
	require.NoError(t, err)
	return p
}

func (f fixture) createProduct(t *testing.T, tenantID uuid.UUID, sku, name string, opts ...productOption) *domain.Product {
	t.Helper()
	p := f.newProduct(t, tenantID, sku, name, opts...)
	require.NoError(t, f.products.Create(t.Context(), p))
	return p
}

func (f fixture) createCategory(t *testing.T, tenantID uuid.UUID, name string, parentID *uuid.UUID) *domain.Category {
	t.Helper()
	c, err := domain.NewCategory(tenantID, name, parentID, now)
	require.NoError(t, err)
	require.NoError(t, f.categories.Create(t.Context(), c))
	return c
}

func names(page domain.ProductPage) []string {
	out := make([]string, 0, len(page.Items))
	for _, p := range page.Items {
		out = append(out, p.Name())
	}
	return out
}
