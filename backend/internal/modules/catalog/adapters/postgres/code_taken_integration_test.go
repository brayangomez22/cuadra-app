//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// requireCodeTaken checks that err is want and names code.
func requireCodeTaken(t *testing.T, err, want error, code string) {
	t.Helper()
	require.ErrorIs(t, err, want)
	var taken *domain.CodeTakenError
	require.ErrorAs(t, err, &taken)
	require.Equal(t, code, taken.Code)
}

func TestCodeTakenNamesTheCode(t *testing.T) {
	f := newFixture(t)

	tests := []struct {
		name, sku, unitCode, unitName string
		barcode, unitBarcode          string
	}{
		{"almacén de ropa", "CAM-M", "docena", "Docena", "7703000000001", "7703000000002"},
		{"ferretería", "TOR-1", "caja", "Caja", "7704000000001", "7704000000002"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("un SKU de otro producto nombra el SKU", func(t *testing.T) {
				tenant := f.createTenant(t)
				f.createProduct(t, tenant, tt.sku, "Primero")

				err := f.products.Create(t.Context(), f.newProduct(t, tenant, tt.sku, "Segundo"))
				requireCodeTaken(t, err, domain.ErrSKUTaken, tt.sku)
			})

			t.Run("un código de barras de otra variante nombra el código", func(t *testing.T) {
				tenant := f.createTenant(t)
				f.createProduct(t, tenant, tt.sku, "Primero", withBarcode(tt.barcode))

				err := f.products.Create(t.Context(), f.newProduct(t, tenant, "OTRO-1", "Segundo", withBarcode(tt.barcode)))
				requireCodeTaken(t, err, domain.ErrBarcodeTaken, tt.barcode)
			})

			t.Run("un código de barras de la unidad de otro producto nombra el código", func(t *testing.T) {
				tenant := f.createTenant(t)
				unit := f.createUnit(t, tenant, tt.unitCode, tt.unitName).Ref()
				first := f.createProduct(t, tenant, tt.sku, "Primero")
				require.NoError(t, f.setUnit(t, first, unit, "12", "5000", tt.unitBarcode))

				second := f.createProduct(t, tenant, "OTRO-1", "Segundo")
				requireCodeTaken(t, f.setUnit(t, second, unit, "6", "3000", tt.unitBarcode), domain.ErrBarcodeTaken, tt.unitBarcode)

				third := f.newProduct(t, tenant, "OTRO-2", "Tercero", withBarcode(tt.unitBarcode))
				requireCodeTaken(t, f.products.Create(t.Context(), third), domain.ErrBarcodeTaken, tt.unitBarcode)
			})

			t.Run("un código de barras de unidad que usa la variante de otro producto nombra el código", func(t *testing.T) {
				tenant := f.createTenant(t)
				unit := f.createUnit(t, tenant, tt.unitCode, tt.unitName).Ref()
				f.createProduct(t, tenant, tt.sku, "Primero", withBarcode(tt.barcode))

				second := f.createProduct(t, tenant, "OTRO-1", "Segundo")
				requireCodeTaken(t, f.setUnit(t, second, unit, "12", "5000", tt.barcode), domain.ErrBarcodeTaken, tt.barcode)
			})
		})
	}
}
