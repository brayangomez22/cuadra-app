//go:build integration

package postgres_test

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// withAxes gives the default variant axis values, so more can be added.
func withAxes(axes map[string]string) productOption {
	return func(p *domain.NewProductParams) { p.Variant.AxisValues = axes }
}

func variant(sku, barcode string, axes map[string]string) domain.VariantParams {
	return domain.VariantParams{SKU: sku, Barcode: barcode, Cost: dec("15000"), Price: dec("30000.5"), AxisValues: axes}
}

// addVariant adds a variant to p and stores the product.
func (f fixture) addVariant(t *testing.T, p *domain.Product, v domain.VariantParams) *domain.ProductVariant {
	t.Helper()
	added, err := p.AddVariant(v, now)
	require.NoError(t, err)
	require.NoError(t, f.products.Update(t.Context(), p))
	return added
}

func TestProductVariants(t *testing.T) {
	f := newFixture(t)

	tests := []struct {
		name          string
		sku, product  string
		first, second map[string]string
	}{
		{"almacén de ropa: talla y color", "CAM-S-ROJO", "Camiseta básica",
			map[string]string{"talla": "S", "color": "Rojo"}, map[string]string{"talla": "M", "color": "Rojo"}},
		{"ferretería: presentación", "PIN-GAL", "Pintura blanca",
			map[string]string{"presentación": "Galón"}, map[string]string{"presentación": "Cuarto"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("guarda y lee un producto con varias variantes", func(t *testing.T) {
				tenant := f.createTenant(t)
				p := f.createProduct(t, tenant, tt.sku, tt.product, withAxes(tt.first))
				added := f.addVariant(t, p, variant(tt.sku+"-2", "7703333333333", tt.second))

				got, err := f.products.GetByID(t.Context(), tenant, p.ID())
				require.NoError(t, err)
				require.Len(t, got.Variants(), 2)
				require.Equal(t, p.DefaultVariant().ID(), got.DefaultVariant().ID())
				require.Equal(t, tt.first, got.DefaultVariant().AxisValues())

				v, err := got.Variant(added.ID())
				require.NoError(t, err)
				require.Equal(t, tt.sku+"-2", v.SKU())
				require.Equal(t, "7703333333333", v.Barcode())
				require.Equal(t, tt.second, v.AxisValues())
				require.True(t, dec("30000.5").Equal(v.Price()))
				require.False(t, v.IsDefault())
				require.Equal(t, now, v.CreatedAt())
			})

			t.Run("la búsqueda encuentra el producto por el SKU y el código de cualquiera de sus variantes", func(t *testing.T) {
				tenant := f.createTenant(t)
				p := f.createProduct(t, tenant, tt.sku, tt.product, withAxes(tt.first))
				f.addVariant(t, p, variant(tt.sku+"-2", "7704444444444", tt.second))

				for _, q := range []string{tt.sku, tt.sku + "-2", "7704444444444"} {
					page, err := f.products.Search(t.Context(), tenant, domain.ProductFilter{Query: q, Limit: 20})
					require.NoError(t, err)
					require.Equal(t, []string{tt.product}, names(page), q)
					require.Len(t, page.Items[0].Variants(), 2, q)
				}
			})
		})
	}

	t.Run("lista productos con sus variantes, la por defecto primero", func(t *testing.T) {
		tenant := f.createTenant(t)
		p := f.createProduct(t, tenant, "CAM-S", "Camiseta", withAxes(map[string]string{"talla": "S"}))
		m := f.addVariant(t, p, variant("CAM-M", "", map[string]string{"talla": "M"}))
		f.createProduct(t, tenant, "CEM-50", "Cemento gris 50kg")

		page, err := f.products.Search(t.Context(), tenant, domain.ProductFilter{Limit: 20})
		require.NoError(t, err)
		require.Equal(t, []string{"Camiseta", "Cemento gris 50kg"}, names(page))
		variants := page.Items[0].Variants()
		require.Len(t, variants, 2)
		require.Equal(t, p.DefaultVariant().ID(), variants[0].ID())
		require.Equal(t, m.ID(), variants[1].ID())
		require.Len(t, page.Items[1].Variants(), 1)
	})

	t.Run("una variante con el SKU o el código de otro producto del tenant se rechaza", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createProduct(t, tenant, "OTRO-1", "Otro producto", withBarcode("7705555555555"))
		p := f.createProduct(t, tenant, "CAM-S", "Camiseta", withAxes(map[string]string{"talla": "S"}))

		_, err := p.AddVariant(variant("otro-1", "", map[string]string{"talla": "M"}), now)
		require.NoError(t, err)
		require.ErrorIs(t, f.products.Update(t.Context(), p), domain.ErrSKUTaken)

		p = f.createProduct(t, tenant, "PAN-S", "Pantalón", withAxes(map[string]string{"talla": "S"}))
		_, err = p.AddVariant(variant("PAN-M", "7705555555555", map[string]string{"talla": "M"}), now)
		require.NoError(t, err)
		require.ErrorIs(t, f.products.Update(t.Context(), p), domain.ErrBarcodeTaken)

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)
		require.Len(t, got.Variants(), 1, "nothing is stored")
	})

	t.Run("el índice único rechaza dos variantes con los mismos ejes aunque se creen a la vez", func(t *testing.T) {
		tenant := f.createTenant(t)
		p := f.createProduct(t, tenant, "CAM-S", "Camiseta", withAxes(map[string]string{"talla": "S"}))

		// Two requests read the product before either stores its new variant,
		// so the domain cannot see the other one: only the database can.
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i := range errs {
			copyOf, err := f.products.GetByID(t.Context(), tenant, p.ID())
			require.NoError(t, err)
			_, err = copyOf.AddVariant(variant("CAM-M-"+string(rune('A'+i)), "", map[string]string{"talla": "M"}), now)
			require.NoError(t, err)
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[i] = f.products.Update(t.Context(), copyOf)
			}()
		}
		wg.Wait()

		failed := 0
		for _, err := range errs {
			if err != nil {
				require.ErrorIs(t, err, domain.ErrDuplicateVariant)
				failed++
			}
		}
		require.Equal(t, 1, failed)
		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)
		require.Len(t, got.Variants(), 2)
	})
}

func TestVariantTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA, tenantB := f.createTenant(t), f.createTenant(t)
	productA := f.createProduct(t, tenantA, "CAM-S", "Camiseta", withAxes(map[string]string{"talla": "S"}))
	variantA := f.addVariant(t, productA, variant("CAM-M", "7706666666666", map[string]string{"talla": "M"}))

	t.Run("el tenant B no encuentra las variantes del tenant A por SKU ni por código", func(t *testing.T) {
		for _, q := range []string{"CAM-M", "7706666666666"} {
			page, err := f.products.Search(t.Context(), tenantB, domain.ProductFilter{Query: q, Limit: 20})
			require.NoError(t, err)
			require.Empty(t, page.Items, q)
		}
	})

	t.Run("el tenant B puede usar el mismo SKU y código en sus variantes", func(t *testing.T) {
		p := f.createProduct(t, tenantB, "CAM-S", "Camiseta", withAxes(map[string]string{"talla": "S"}))
		f.addVariant(t, p, variant("CAM-M", "7706666666666", map[string]string{"talla": "M"}))
	})

	t.Run("el tenant B no agrega ni modifica variantes de un producto del tenant A", func(t *testing.T) {
		forged, err := domain.RehydrateProduct(domain.ProductSnapshot{
			ID: productA.ID(), TenantID: tenantB, Name: "Camiseta", BaseUnit: domain.UnitPiece,
			TaxRate: productA.TaxRate(), Active: true, CreatedAt: now,
			Variants: []domain.VariantSnapshot{
				{ID: productA.DefaultVariant().ID(), SKU: "HACK-1", Cost: dec("1"), Price: dec("1"),
					AxisValues: map[string]string{"talla": "S"}, IsDefault: true, Active: true, CreatedAt: now},
				{ID: variantA.ID(), SKU: "HACK-2", Cost: dec("1"), Price: dec("1"),
					AxisValues: map[string]string{"talla": "M"}, Active: true, CreatedAt: now},
				{ID: uuid.Must(uuid.NewV7()), SKU: "HACK-3", Cost: dec("1"), Price: dec("1"),
					AxisValues: map[string]string{"talla": "L"}, Active: true, CreatedAt: now},
			},
		})
		require.NoError(t, err)

		require.ErrorIs(t, f.products.Update(t.Context(), forged), domain.ErrProductNotFound)

		got, err := f.products.GetByID(t.Context(), tenantA, productA.ID())
		require.NoError(t, err)
		require.Len(t, got.Variants(), 2)
		v, err := got.Variant(variantA.ID())
		require.NoError(t, err)
		require.Equal(t, "CAM-M", v.SKU())
	})
}
