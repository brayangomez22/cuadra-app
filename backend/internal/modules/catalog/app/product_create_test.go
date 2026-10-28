package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// withVariants is a product of the actor's tenant, in meters, with
// variants (the first one is its default).
func (e *env) withVariants(t *testing.T, name string, variants ...app.VariantInput) app.ProductWithVariantsInput {
	t.Helper()
	in := e.productInput(t)
	return app.ProductWithVariantsInput{
		Name: name, BaseUnitID: in.BaseUnitID, TaxRate: in.TaxRate, Variants: variants,
	}
}

func TestCreateProductWithVariants(t *testing.T) {
	tests := []struct {
		name     string
		product  string
		variants []app.VariantInput
	}{
		{"almacén de ropa: tallas y colores", "Camiseta básica", []app.VariantInput{
			variantInput("cam-s-rojo", map[string]string{"talla": "S", "color": "Rojo"}),
			variantInput("cam-m-rojo", map[string]string{"talla": "M", "color": "Rojo"}),
			variantInput("cam-m-azul", map[string]string{"talla": "M", "color": "Azul"}),
		}},
		{"ferretería: presentaciones", "Pintura vinilo blanca", []app.VariantInput{
			variantInput("pin-galon", map[string]string{"presentación": "Galón"}),
			variantInput("pin-cuarto", map[string]string{"presentación": "Cuarto"}),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("crea un producto con sus N variantes en una sola operación", func(t *testing.T) {
				e := newEnv(t)

				p, err := e.svc.CreateProductWithVariants(t.Context(), e.actor, e.withVariants(t, tt.product, tt.variants...))
				require.NoError(t, err)

				stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
				require.NoError(t, err)
				require.Len(t, stored.Variants(), len(tt.variants))
				require.True(t, stored.Variants()[0].IsDefault())
				for i, v := range stored.Variants() {
					require.Equal(t, tt.variants[i].AxisValues, v.AxisValues())
				}
				require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.products_created", "", ""))
				_, attrs := e.span(t, "catalog.CreateProductWithVariants")
				require.Equal(t, p.ID().String(), attrs["product.id"].AsString())
			})

			t.Run("si una variante no es válida no guarda nada", func(t *testing.T) {
				e := newEnv(t)
				variants := append([]app.VariantInput{}, tt.variants...)
				variants = append(variants, variantInput("repetida", tt.variants[0].AxisValues))

				_, err := e.svc.CreateProductWithVariants(t.Context(), e.actor, e.withVariants(t, tt.product, variants...))
				require.ErrorIs(t, err, domain.ErrDuplicateVariant)
				require.Empty(t, e.store.products)
			})
		})
	}

	t.Run("rechaza un producto sin variantes", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.CreateProductWithVariants(t.Context(), e.actor, e.withVariants(t, "Camiseta básica"))
		require.ErrorIs(t, err, domain.ErrNoVariants)
	})

	t.Run("rechaza un valor que no está en los ejes de la empresa", func(t *testing.T) {
		e := newEnv(t)
		in := e.withVariants(t, "Camiseta básica",
			variantInput("cam-s", map[string]string{"talla": "S"}),
			variantInput("cam-xxl", map[string]string{"talla": "XXL"}))

		_, err := e.svc.CreateProductWithVariants(t.Context(), e.actor, in)
		require.ErrorIs(t, err, domain.ErrAxisValueNotAllowed)
		require.Empty(t, e.store.products)
	})
}
