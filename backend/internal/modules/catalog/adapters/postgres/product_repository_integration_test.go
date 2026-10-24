//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestProductRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lee un producto con 4 decimales sin pérdida", func(t *testing.T) {
		tenant := f.createTenant(t)
		category := f.createCategory(t, tenant, "Plomería", nil)
		p := f.createProduct(t, tenant, "tub-pvc-12", "Tubo PVC presión 1/2 pulgada", withBarcode("7701234567890"), withCategory(category.ID()))

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)

		require.Equal(t, p.ID(), got.ID())
		v := got.DefaultVariant()
		require.Equal(t, p.DefaultVariant().ID(), v.ID())
		require.Equal(t, p.ID(), v.ProductID())
		require.Equal(t, "TUB-PVC-12", v.SKU())
		require.Equal(t, "7701234567890", v.Barcode())
		require.Equal(t, "Tubo PVC presión 1/2 pulgada", got.Name())
		require.Equal(t, "Producto de prueba", got.Description())
		require.Equal(t, p.CategoryID(), got.CategoryID())
		require.Equal(t, f.piece[tenant], got.BaseUnit())
		require.True(t, dec("8333.3333").Equal(v.Cost()), v.Cost().String())
		require.True(t, dec("10000.1234").Equal(v.Price()), v.Price().String())
		require.True(t, v.IsDefault())
		require.True(t, v.IsActive())
		require.Empty(t, v.AxisValues())
		require.Equal(t, now, v.CreatedAt())
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
		meters := f.createUnit(t, tenant, "m", "Metro").Ref()
		require.NoError(t, p.UpdateDetails(domain.ProductDetails{
			Name: "Cable 14 AWG", CategoryID: &categoryID, BaseUnit: meters, TaxRate: rate,
		}))
		id := p.DefaultVariant().ID()
		require.NoError(t, p.UpdateVariant(id, domain.VariantDetails{SKU: "CAB-14", Cost: dec("1200.5")}))
		require.NoError(t, p.ChangeVariantPrice(id, dec("2500")))
		require.NoError(t, p.SetVariantActive(id, false))
		p.Deactivate()

		require.NoError(t, f.products.Update(t.Context(), p))

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)
		v := got.DefaultVariant()
		require.Equal(t, "CAB-14", v.SKU())
		require.Empty(t, v.Barcode())
		require.True(t, dec("1200.5").Equal(v.Cost()))
		require.True(t, dec("2500").Equal(v.Price()))
		require.False(t, v.IsActive())
		require.Equal(t, "Cable 14 AWG", got.Name())
		require.Equal(t, &categoryID, got.CategoryID())
		require.Equal(t, meters, got.BaseUnit())
		require.True(t, dec("0.05").Equal(got.TaxRate().Rate()))
		require.False(t, got.IsActive())
	})

	t.Run("actualizar con un SKU de otro producto devuelve ErrSKUTaken", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createProduct(t, tenant, "A-1", "Producto A")
		b := f.createProduct(t, tenant, "B-1", "Producto B")
		require.NoError(t, b.UpdateVariant(b.DefaultVariant().ID(), domain.VariantDetails{SKU: "A-1", Cost: dec("1")}))

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

	t.Run("filtra por ids, con o sin texto", func(t *testing.T) {
		cement := search(t, domain.ProductFilter{Query: "cem-50"}).Items[0]
		pipe := search(t, domain.ProductFilter{Query: "tub-12"}).Items[0]
		ids := []uuid.UUID{pipe.ID(), cement.ID(), inactive.ID()}

		require.Equal(t, []string{"Cemento gris 50kg", "Tubo PVC descontinuado", "Tubo PVC presión 1/2 pulgada"},
			names(search(t, domain.ProductFilter{IDs: ids})))
		require.Equal(t, 3, search(t, domain.ProductFilter{IDs: ids}).Total)
		require.Equal(t, []string{"Tubo PVC presión 1/2 pulgada"},
			names(search(t, domain.ProductFilter{Query: "tubo", Active: &active, IDs: ids})))
		require.Empty(t, search(t, domain.ProductFilter{IDs: []uuid.UUID{}}).Items)
	})

	t.Run("aislamiento: los ids de otro tenant no aparecen", func(t *testing.T) {
		other := f.createTenant(t)
		foreign := f.createProduct(t, other, "AJENO-1", "Producto de otra ferretería")
		pipe := search(t, domain.ProductFilter{Query: "tub-12"}).Items[0]

		page := search(t, domain.ProductFilter{IDs: []uuid.UUID{foreign.ID(), pipe.ID()}})

		require.Equal(t, []string{"Tubo PVC presión 1/2 pulgada"}, names(page))
		require.Equal(t, 1, page.Total)
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
