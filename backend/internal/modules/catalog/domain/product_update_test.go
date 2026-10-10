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
		SKU:         "  tub-pvc-12 ",
		Barcode:     " 7709876543210 ",
		Name:        "  Tubo PVC presión 1/2 pulgada ",
		Description: " Tramo de 6 m ",
		CategoryID:  &categoryID,
		BaseUnit:    domain.UnitMeter,
		Cost:        dec("3500.1234"),
		TaxRate:     mustTaxRate(t, "0.05"),
	}
}

func TestProductUpdateDetails(t *testing.T) {
	t.Run("actualiza los datos de un producto y normaliza el SKU", func(t *testing.T) {
		product := mustProduct(t, "28000", "32500")
		d := validDetails(t)

		require.NoError(t, product.UpdateDetails(d))

		require.Equal(t, "TUB-PVC-12", product.SKU())
		require.Equal(t, "7709876543210", product.Barcode())
		require.Equal(t, "Tubo PVC presión 1/2 pulgada", product.Name())
		require.Equal(t, "Tramo de 6 m", product.Description())
		require.Equal(t, d.CategoryID, product.CategoryID())
		require.Equal(t, domain.UnitMeter, product.BaseUnit())
		require.True(t, dec("3500.1234").Equal(product.Cost()))
		require.True(t, dec("0.05").Equal(product.TaxRate().Rate()))
		require.True(t, dec("32500").Equal(product.Price()), "the price changes only through ChangePrice")
	})

	t.Run("quita el código de barras, la descripción y la categoría", func(t *testing.T) {
		product := mustProduct(t, "28000", "32500")
		d := validDetails(t)
		d.Barcode, d.Description, d.CategoryID = "", "", nil

		require.NoError(t, product.UpdateDetails(d))

		require.Empty(t, product.Barcode())
		require.Empty(t, product.Description())
		require.Nil(t, product.CategoryID())
	})

	t.Run("un producto inactivo también puede actualizar sus datos", func(t *testing.T) {
		product := mustProduct(t, "28000", "32500")
		product.Deactivate()

		require.NoError(t, product.UpdateDetails(validDetails(t)))
		require.Equal(t, "TUB-PVC-12", product.SKU())
		require.False(t, product.IsActive())
	})

	t.Run("rechaza actualización con datos inválidos sin modificar el producto", func(t *testing.T) {
		nilCategory := uuid.Nil
		tests := []struct {
			name   string
			change func(*domain.ProductDetails)
			want   error
		}{
			{"SKU vacío", func(d *domain.ProductDetails) { d.SKU = " " }, domain.ErrInvalidSKU},
			{"código de barras con espacios", func(d *domain.ProductDetails) { d.Barcode = "770 123" }, domain.ErrInvalidBarcode},
			{"nombre vacío", func(d *domain.ProductDetails) { d.Name = "" }, domain.ErrInvalidProductName},
			{"descripción muy larga", func(d *domain.ProductDetails) { d.Description = strings.Repeat("a", 2001) }, domain.ErrInvalidDescription},
			{"categoría nula", func(d *domain.ProductDetails) { d.CategoryID = &nilCategory }, domain.ErrInvalidCategoryID},
			{"unidad desconocida", func(d *domain.ProductDetails) { d.BaseUnit = "docena" }, domain.ErrInvalidUnit},
			{"costo negativo", func(d *domain.ProductDetails) { d.Cost = dec("-1") }, domain.ErrInvalidCost},
			{"costo con 5 decimales", func(d *domain.ProductDetails) { d.Cost = dec("1.00001") }, domain.ErrInvalidCost},
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
