package domain_test

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
	tests := []struct {
		name     string
		axes     map[string]string
		newAxes  map[string]string
		unit     domain.UnitRef
		unitCode string
	}{
		{
			name:     "almacén de ropa: talla",
			axes:     map[string]string{"talla": "S"},
			newAxes:  map[string]string{"talla": "M"},
			unit:     dozens,
			unitCode: "7709000000001",
		},
		{
			name:     "ferretería: presentación",
			axes:     map[string]string{"presentación": "Galón"},
			newAxes:  map[string]string{"presentación": "Cuarto"},
			unit:     box,
			unitCode: "7709000000002",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("el SKU repetido entre variantes nombra el SKU", func(t *testing.T) {
				product := productWithAxes(t, tt.axes)

				_, err := product.AddVariant(variantParams("cem-gris-50", tt.newAxes), testNow)
				requireCodeTaken(t, err, domain.ErrSKUTaken, "CEM-GRIS-50")
			})

			t.Run("el código de barras repetido entre variantes nombra el código", func(t *testing.T) {
				product := productWithAxes(t, tt.axes)
				params := variantParams("VAR-2", tt.newAxes)
				params.Barcode = "7701234567890"

				_, err := product.AddVariant(params, testNow)
				requireCodeTaken(t, err, domain.ErrBarcodeTaken, "7701234567890")
			})

			t.Run("el código de una unidad que ya usa otra variante nombra el código", func(t *testing.T) {
				product := productWithAxes(t, tt.axes)
				variant, err := product.AddVariant(variantParams("VAR-2", tt.newAxes), testNow)
				require.NoError(t, err)
				unit := domain.VariantUnitParams{Unit: tt.unit, Factor: dec("12"), Price: dec("1000"), Barcode: tt.unitCode}
				require.NoError(t, product.SetVariantUnit(product.DefaultVariant().ID(), unit))

				err = product.SetVariantUnit(variant.ID(), unit)
				requireCodeTaken(t, err, domain.ErrBarcodeTaken, tt.unitCode)
			})
		})
	}
}
