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
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

type fixture struct {
	db         *db.DB
	products   *postgres.ProductRepository
	categories *postgres.CategoryRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{db: d, products: postgres.NewProductRepository(d), categories: postgres.NewCategoryRepository(d)}
}

// createTenant stores a tenant row directly: the catalog does not depend on
// the identity module, but its tables reference tenants.
func (f fixture) createTenant(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	require.NoError(t, f.db.WithTenantTx(t.Context(), id, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, created_at) VALUES ($1, 'Ferretería', '890903938-8', 'active', $2)", id, now)
		return err
	}))
	return id
}

type productOption func(*domain.NewProductParams)

func withBarcode(code string) productOption {
	return func(p *domain.NewProductParams) { p.Barcode = code }
}

func withCategory(id uuid.UUID) productOption {
	return func(p *domain.NewProductParams) { p.CategoryID = &id }
}

func (f fixture) newProduct(t *testing.T, tenantID uuid.UUID, sku, name string, opts ...productOption) *domain.Product {
	t.Helper()
	rate, err := domain.ParseTaxRate(dec("0.19"))
	require.NoError(t, err)
	params := domain.NewProductParams{
		TenantID: tenantID, SKU: sku, Name: name, Description: "Producto de prueba",
		BaseUnit: domain.UnitPiece, Cost: dec("8333.3333"), Price: dec("10000.1234"), TaxRate: rate,
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

func TestProductRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lee un producto con 4 decimales sin pérdida", func(t *testing.T) {
		tenant := f.createTenant(t)
		category := f.createCategory(t, tenant, "Plomería", nil)
		p := f.createProduct(t, tenant, "tub-pvc-12", "Tubo PVC presión 1/2 pulgada", withBarcode("7701234567890"), withCategory(category.ID()))

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)

		require.Equal(t, p.ID(), got.ID())
		require.Equal(t, "TUB-PVC-12", got.SKU())
		require.Equal(t, "7701234567890", got.Barcode())
		require.Equal(t, "Tubo PVC presión 1/2 pulgada", got.Name())
		require.Equal(t, "Producto de prueba", got.Description())
		require.Equal(t, p.CategoryID(), got.CategoryID())
		require.Equal(t, domain.UnitPiece, got.BaseUnit())
		require.True(t, dec("8333.3333").Equal(got.Cost()), got.Cost().String())
		require.True(t, dec("10000.1234").Equal(got.Price()), got.Price().String())
		require.True(t, dec("0.19").Equal(got.TaxRate().Rate()))
		require.True(t, got.IsActive())
		require.Equal(t, now, got.CreatedAt())
	})

	t.Run("devuelve ErrProductNotFound si no existe", func(t *testing.T) {
		tenant := f.createTenant(t)
		_, err := f.products.GetByID(t.Context(), tenant, uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("rechaza SKU duplicado en el mismo tenant y lo permite en otro tenant", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		f.createProduct(t, tenantA, "CEM-50", "Cemento gris 50kg")

		err := f.products.Create(t.Context(), f.newProduct(t, tenantA, "cem-50", "Otro cemento"))
		require.ErrorIs(t, err, domain.ErrSKUTaken)

		require.NoError(t, f.products.Create(t.Context(), f.newProduct(t, tenantB, "CEM-50", "Cemento gris 50kg")))
	})

	t.Run("rechaza código de barras duplicado en el mismo tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createProduct(t, tenant, "A-1", "Producto A", withBarcode("7701111111111"))

		err := f.products.Create(t.Context(), f.newProduct(t, tenant, "A-2", "Producto B", withBarcode("7701111111111")))
		require.ErrorIs(t, err, domain.ErrBarcodeTaken)
	})

	t.Run("permite varios productos sin código de barras", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createProduct(t, tenant, "SIN-1", "Producto sin código 1")
		f.createProduct(t, tenant, "SIN-2", "Producto sin código 2")
	})

	t.Run("rechaza un producto con categoría inexistente", func(t *testing.T) {
		tenant := f.createTenant(t)
		err := f.products.Create(t.Context(), f.newProduct(t, tenant, "X-1", "Producto", withCategory(uuid.Must(uuid.NewV7()))))
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rechaza un producto que apunta a una categoría de otro tenant", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		categoryB := f.createCategory(t, tenantB, "Plomería", nil)

		err := f.products.Create(t.Context(), f.newProduct(t, tenantA, "X-1", "Producto", withCategory(categoryB.ID())))
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("actualiza todos los datos de un producto", func(t *testing.T) {
		tenant := f.createTenant(t)
		category := f.createCategory(t, tenant, "Eléctricos", nil)
		p := f.createProduct(t, tenant, "CAB-12", "Cable", withBarcode("7702222222222"))
		rate, err := domain.ParseTaxRate(dec("0.05"))
		require.NoError(t, err)
		categoryID := category.ID()
		require.NoError(t, p.UpdateDetails(domain.ProductDetails{
			SKU: "CAB-14", Name: "Cable 14 AWG", CategoryID: &categoryID,
			BaseUnit: domain.UnitMeter, Cost: dec("1200.5"), TaxRate: rate,
		}))
		require.NoError(t, p.ChangePrice(dec("2500")))
		p.Deactivate()

		require.NoError(t, f.products.Update(t.Context(), p))

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)
		require.Equal(t, "CAB-14", got.SKU())
		require.Empty(t, got.Barcode())
		require.Equal(t, "Cable 14 AWG", got.Name())
		require.Equal(t, &categoryID, got.CategoryID())
		require.Equal(t, domain.UnitMeter, got.BaseUnit())
		require.True(t, dec("1200.5").Equal(got.Cost()))
		require.True(t, dec("2500").Equal(got.Price()))
		require.True(t, dec("0.05").Equal(got.TaxRate().Rate()))
		require.False(t, got.IsActive())
	})

	t.Run("actualizar con un SKU de otro producto devuelve ErrSKUTaken", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createProduct(t, tenant, "A-1", "Producto A")
		b := f.createProduct(t, tenant, "B-1", "Producto B")
		require.NoError(t, b.UpdateDetails(domain.ProductDetails{
			SKU: "A-1", Name: b.Name(), BaseUnit: b.BaseUnit(), Cost: b.Cost(), TaxRate: b.TaxRate(),
		}))

		require.ErrorIs(t, f.products.Update(t.Context(), b), domain.ErrSKUTaken)
	})

	t.Run("actualizar un producto inexistente devuelve ErrProductNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		require.ErrorIs(t, f.products.Update(t.Context(), f.newProduct(t, tenant, "A-1", "Producto")), domain.ErrProductNotFound)
	})
}

func TestProductSearch(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)
	plumbing := f.createCategory(t, tenant, "Plomería", nil)
	f.createProduct(t, tenant, "TUB-12", "Tubo PVC presión 1/2 pulgada", withCategory(plumbing.ID()))
	f.createProduct(t, tenant, "TUB-34", "Tubo PVC sanitario 3 pulgadas", withCategory(plumbing.ID()))
	f.createProduct(t, tenant, "CEM-50", "Cemento gris 50kg", withBarcode("7701234567890"))
	f.createProduct(t, tenant, "PVC", "Pegante para tubería")
	inactive := f.createProduct(t, tenant, "TUB-OLD", "Tubo PVC descontinuado")
	inactive.Deactivate()
	require.NoError(t, f.products.Update(t.Context(), inactive))
	active := true

	search := func(t *testing.T, filter domain.ProductFilter) domain.ProductPage {
		t.Helper()
		if filter.Limit == 0 {
			filter.Limit = 20
		}
		page, err := f.products.Search(t.Context(), tenant, filter)
		require.NoError(t, err)
		return page
	}

	t.Run("busca 'tubo pvc' y encuentra 'Tubo PVC presión 1/2 pulgada'", func(t *testing.T) {
		page := search(t, domain.ProductFilter{Query: "tubo pvc", Active: &active})
		require.Contains(t, names(page), "Tubo PVC presión 1/2 pulgada")
		require.NotContains(t, names(page), "Cemento gris 50kg")
	})

	t.Run("cada palabra debe aparecer, en cualquier orden", func(t *testing.T) {
		page := search(t, domain.ProductFilter{Query: "pulgada tubo 1/2", Active: &active})
		require.Equal(t, []string{"Tubo PVC presión 1/2 pulgada"}, names(page))
	})

	t.Run("busca sin tildes: 'presion' encuentra 'presión'", func(t *testing.T) {
		page := search(t, domain.ProductFilter{Query: "PRESION", Active: &active})
		require.Equal(t, []string{"Tubo PVC presión 1/2 pulgada"}, names(page))
	})

	t.Run("encuentra por SKU exacto y por código de barras", func(t *testing.T) {
		require.Equal(t, []string{"Cemento gris 50kg"}, names(search(t, domain.ProductFilter{Query: "cem-50"})))
		require.Equal(t, []string{"Cemento gris 50kg"}, names(search(t, domain.ProductFilter{Query: "7701234567890"})))
	})

	t.Run("el SKU exacto sale antes que las coincidencias por nombre", func(t *testing.T) {
		page := search(t, domain.ProductFilter{Query: "pvc", Active: &active})
		require.Equal(t, "Pegante para tubería", names(page)[0])
		require.Len(t, page.Items, 3)
	})

	t.Run("los comodines de LIKE se buscan como texto", func(t *testing.T) {
		require.Empty(t, search(t, domain.ProductFilter{Query: "%"}).Items)
		require.Empty(t, search(t, domain.ProductFilter{Query: "tubo_pvc"}).Items)
	})

	t.Run("filtra por estado y categoría", func(t *testing.T) {
		inactiveOnly := false
		require.Equal(t, []string{"Tubo PVC descontinuado"}, names(search(t, domain.ProductFilter{Query: "tubo", Active: &inactiveOnly})))
		require.Len(t, search(t, domain.ProductFilter{Query: "tubo"}).Items, 3)

		categoryID := plumbing.ID()
		page := search(t, domain.ProductFilter{CategoryID: &categoryID})
		require.Equal(t, []string{"Tubo PVC presión 1/2 pulgada", "Tubo PVC sanitario 3 pulgadas"}, names(page))
	})

	t.Run("sin texto lista por nombre", func(t *testing.T) {
		page := search(t, domain.ProductFilter{Active: &active})
		require.Equal(t, []string{
			"Cemento gris 50kg", "Pegante para tubería", "Tubo PVC presión 1/2 pulgada", "Tubo PVC sanitario 3 pulgadas",
		}, names(page))
	})

	t.Run("pagina resultados con limit y offset y devuelve el total", func(t *testing.T) {
		page := search(t, domain.ProductFilter{Active: &active, Limit: 2, Offset: 1})
		require.Equal(t, []string{"Pegante para tubería", "Tubo PVC presión 1/2 pulgada"}, names(page))
		require.Equal(t, 4, page.Total)

		page = search(t, domain.ProductFilter{Query: "tubo", Active: &active, Limit: 1, Offset: 5})
		require.Empty(t, page.Items)
		require.Equal(t, 2, page.Total)
	})
}

func TestCategoryRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda, lista por nombre y lee categorías", func(t *testing.T) {
		tenant := f.createTenant(t)
		plumbing := f.createCategory(t, tenant, "Plomería", nil)
		plumbingID := plumbing.ID()
		pipes := f.createCategory(t, tenant, "Tubería", &plumbingID)
		f.createCategory(t, tenant, "Eléctricos", nil)

		list, err := f.categories.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Len(t, list, 3)
		require.Equal(t, "Eléctricos", list[0].Name())

		got, err := f.categories.GetByID(t.Context(), tenant, pipes.ID())
		require.NoError(t, err)
		require.Equal(t, "Tubería", got.Name())
		require.Equal(t, &plumbingID, got.ParentID())
		require.Equal(t, now, got.CreatedAt())
	})

	t.Run("devuelve ErrCategoryNotFound si no existe", func(t *testing.T) {
		tenant := f.createTenant(t)
		_, err := f.categories.GetByID(t.Context(), tenant, uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rechaza un padre inexistente o de otro tenant", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		parentB := f.createCategory(t, tenantB, "Plomería", nil)
		for _, parentID := range []uuid.UUID{uuid.Must(uuid.NewV7()), parentB.ID()} {
			c, err := domain.NewCategory(tenantA, "Tubería", &parentID, now)
			require.NoError(t, err)
			require.ErrorIs(t, f.categories.Create(t.Context(), c), domain.ErrCategoryNotFound)
		}
	})

	t.Run("renombra y mueve una categoría", func(t *testing.T) {
		tenant := f.createTenant(t)
		a := f.createCategory(t, tenant, "A", nil)
		b := f.createCategory(t, tenant, "B", nil)
		aID := a.ID()
		require.NoError(t, b.Rename("B2"))
		require.NoError(t, b.MoveTo(&aID))

		require.NoError(t, f.categories.Update(t.Context(), b))

		got, err := f.categories.GetByID(t.Context(), tenant, b.ID())
		require.NoError(t, err)
		require.Equal(t, "B2", got.Name())
		require.Equal(t, &aID, got.ParentID())
	})

	t.Run("devuelve los ancestros del más cercano al más lejano", func(t *testing.T) {
		tenant := f.createTenant(t)
		a := f.createCategory(t, tenant, "A", nil)
		aID := a.ID()
		b := f.createCategory(t, tenant, "B", &aID)
		bID := b.ID()
		c := f.createCategory(t, tenant, "C", &bID)

		chain, err := f.categories.Ancestors(t.Context(), tenant, c.ID())
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{c.ID(), bID, aID}, chain)

		none, err := f.categories.Ancestors(t.Context(), tenant, uuid.Must(uuid.NewV7()))
		require.NoError(t, err)
		require.Empty(t, none)
	})

	t.Run("rechaza borrar una categoría con productos o subcategorías", func(t *testing.T) {
		tenant := f.createTenant(t)
		parent := f.createCategory(t, tenant, "Plomería", nil)
		parentID := parent.ID()
		f.createCategory(t, tenant, "Tubería", &parentID)
		withProduct := f.createCategory(t, tenant, "Grifería", nil)
		f.createProduct(t, tenant, "GRI-1", "Grifo", withCategory(withProduct.ID()))

		require.ErrorIs(t, f.categories.Delete(t.Context(), tenant, parentID), domain.ErrCategoryInUse)
		require.ErrorIs(t, f.categories.Delete(t.Context(), tenant, withProduct.ID()), domain.ErrCategoryInUse)
	})

	t.Run("borra una categoría vacía", func(t *testing.T) {
		tenant := f.createTenant(t)
		c := f.createCategory(t, tenant, "Plomería", nil)

		require.NoError(t, f.categories.Delete(t.Context(), tenant, c.ID()))
		_, err := f.categories.GetByID(t.Context(), tenant, c.ID())
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)
		require.ErrorIs(t, f.categories.Delete(t.Context(), tenant, c.ID()), domain.ErrCategoryNotFound)
	})
}

func TestCatalogTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA, tenantB := f.createTenant(t), f.createTenant(t)
	categoryA := f.createCategory(t, tenantA, "Plomería", nil)
	productA := f.createProduct(t, tenantA, "TUB-12", "Tubo PVC presión 1/2 pulgada", withCategory(categoryA.ID()))

	t.Run("el tenant B no ve los productos ni las categorías del tenant A", func(t *testing.T) {
		_, err := f.products.GetByID(t.Context(), tenantB, productA.ID())
		require.ErrorIs(t, err, domain.ErrProductNotFound)
		_, err = f.categories.GetByID(t.Context(), tenantB, categoryA.ID())
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)

		list, err := f.categories.List(t.Context(), tenantB)
		require.NoError(t, err)
		require.Empty(t, list)
		chain, err := f.categories.Ancestors(t.Context(), tenantB, categoryA.ID())
		require.NoError(t, err)
		require.Empty(t, chain)
	})

	t.Run("el tenant B no encuentra los productos del tenant A al buscar", func(t *testing.T) {
		for _, q := range []string{"", "tubo", "TUB-12"} {
			page, err := f.products.Search(t.Context(), tenantB, domain.ProductFilter{Query: q, Limit: 20})
			require.NoError(t, err)
			require.Empty(t, page.Items, q)
			require.Zero(t, page.Total, q)
		}
	})

	t.Run("el tenant B no edita ni borra datos del tenant A", func(t *testing.T) {
		// The same entities, but claiming to belong to tenant B.
		forged, err := domain.RehydrateProduct(domain.ProductSnapshot{
			ID: productA.ID(), TenantID: tenantB, SKU: "HACK", Name: "Cambiado", BaseUnit: domain.UnitPiece,
			Cost: dec("1"), Price: dec("1"), TaxRate: productA.TaxRate(), Active: true, CreatedAt: now,
		})
		require.NoError(t, err)
		require.ErrorIs(t, f.products.Update(t.Context(), forged), domain.ErrProductNotFound)

		forgedCategory, err := domain.RehydrateCategory(domain.CategorySnapshot{
			ID: categoryA.ID(), TenantID: tenantB, Name: "Cambiada", CreatedAt: now,
		})
		require.NoError(t, err)
		require.ErrorIs(t, f.categories.Update(t.Context(), forgedCategory), domain.ErrCategoryNotFound)
		require.ErrorIs(t, f.categories.Delete(t.Context(), tenantB, categoryA.ID()), domain.ErrCategoryNotFound)

		got, err := f.products.GetByID(t.Context(), tenantA, productA.ID())
		require.NoError(t, err)
		require.Equal(t, "TUB-12", got.SKU())
		gotCategory, err := f.categories.GetByID(t.Context(), tenantA, categoryA.ID())
		require.NoError(t, err)
		require.Equal(t, "Plomería", gotCategory.Name())
	})
}
