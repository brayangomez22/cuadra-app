package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestNewTaxRate(t *testing.T) {
	t.Run("guarda el tratamiento del IVA con su tasa", func(t *testing.T) {
		tests := []struct {
			name      string
			rate      string
			treatment vat.Treatment
			want      vat.Treatment
		}{
			{"el producto con tasa 0 es excluido si no se dice (medicamento de droguería)", "0", "", vat.Excluded},
			{"el producto con tasa positiva es gravado si no se dice (pintura de ferretería)", "0.19", "", vat.Taxed},
			{"un producto exento queda exento (cuaderno de papelería)", "0", vat.Exempt, vat.Exempt},
			{"un producto gravado al 5 % (almacén de ropa)", "0.05", vat.Taxed, vat.Taxed},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rate, err := domain.NewTaxRate(dec(tt.rate), tt.treatment)
				require.NoError(t, err)
				require.True(t, dec(tt.rate).Equal(rate.Rate()))
				require.Equal(t, tt.want, rate.Treatment())
			})
		}
	})

	t.Run("rechaza un tratamiento que no corresponde a la tasa", func(t *testing.T) {
		for _, tt := range []struct {
			rate      string
			treatment vat.Treatment
		}{{"0", vat.Taxed}, {"0.19", vat.Exempt}, {"0.05", vat.Excluded}, {"0", "zero_rated"}} {
			_, err := domain.NewTaxRate(dec(tt.rate), tt.treatment)
			require.ErrorIs(t, err, domain.ErrInvalidTaxTreatment, "%s %s", tt.rate, tt.treatment)
		}
	})

	t.Run("una tasa inválida sigue siendo ErrInvalidTaxRate", func(t *testing.T) {
		_, err := domain.NewTaxRate(dec("1"), vat.Taxed)
		require.ErrorIs(t, err, domain.ErrInvalidTaxRate)
	})

	t.Run("la tasa cero por defecto es excluida", func(t *testing.T) {
		var rate domain.TaxRate
		require.Equal(t, vat.Excluded, rate.Treatment())
	})
}
