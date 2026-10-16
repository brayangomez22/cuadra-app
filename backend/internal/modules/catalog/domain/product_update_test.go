package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func validDetails(t *testing.T) domain.ProductDetails {
	t.Helper()
	categoryID := uuid.Must(uuid.NewV7())
	return domain.ProductDetails{
		Name:        "  Tubo PVC presión 1/2 pulgada ",
		Description: " Tramo de 6 m ",
		CategoryID:  &categoryID,
		BaseUnit:    meters,
		TaxRate:     mustTaxRate(t, "0.05"),
	}
}

func TestProductUpdateDetails(t *testing.T) {
	t.Run("actualiza los datos de un producto sin tocar sus variantes", func(t *testing.T) {
		product := mustProduct(t, "28000", "32500")
		variant := *product.DefaultVariant()
		d := validDetails(t)

		require.NoError(t, product.UpdateDetails(d))

		require.Equal(t, "Tubo PVC presión 1/2 pulgada", product.Name())
		require.Equal(t, "Tramo de 6 m", product.Description())
		require.Equal(t, d.CategoryID, product.CategoryID())
		require.Equal(t, meters, product.BaseUnit())
		require.True(t, dec("0.05").Equal(product.TaxRate().Rate()))
		require.Equal(t, variant, *product.DefaultVariant())
	})

	t.Run("quita la descripción y la categoría", func(t *testing.T) {
		product := mustProduct(t, "28000", "32500")
		d := validDetails(t)
		d.Description, d.CategoryID = "", nil

		require.NoError(t, product.UpdateDetails(d))

		require.Empty(t, product.Description())
		require.Nil(t, product.CategoryID())
	})

	t.Run("un producto inactivo también puede actualizar sus datos", func(t *testing.T) {
		product := mustProduct(t, "28000", "32500")
		product.Deactivate()

		require.NoError(t, product.UpdateDetails(validDetails(t)))
		require.Equal(t, "Tubo PVC presión 1/2 pulgada", product.Name())
		require.False(t, product.IsActive())
	})

	t.Run("rechaza actualización con datos inválidos sin modificar el producto", func(t *testing.T) {
		nilCategory := uuid.Nil
		tests := []struct {
			name   string
			change func(*domain.ProductDetails)
			want   error
		}{
			{"nombre vacío", func(d *domain.ProductDetails) { d.Name = "" }, domain.ErrInvalidProductName},
			{"descripción muy larga", func(d *domain.ProductDetails) { d.Description = strings.Repeat("a", 2001) }, domain.ErrInvalidDescription},
			{"categoría nula", func(d *domain.ProductDetails) { d.CategoryID = &nilCategory }, domain.ErrInvalidCategoryID},
			{"sin unidad base", func(d *domain.ProductDetails) { d.BaseUnit = domain.UnitRef{} }, domain.ErrInvalidUnit},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				product := mustProduct(t, "28000", "32500")
				before := *product
				d := validDetails(t)
				tt.change(&d)

				require.ErrorIs(t, product.UpdateDetails(d), tt.want)
				require.Equal(t, before, *product)
			})
		}
	})
}
