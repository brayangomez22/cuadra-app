package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestParseTaxRate(t *testing.T) {
	t.Run("acepta tasas de IVA 0, 5% y 19%", func(t *testing.T) {
		for _, s := range []string{"0", "0.05", "0.19"} {
			rate, err := domain.ParseTaxRate(dec(s))
			require.NoError(t, err)
			require.True(t, dec(s).Equal(rate.Rate()))
		}
	})

	t.Run("la tasa cero por defecto es válida", func(t *testing.T) {
		var rate domain.TaxRate
		require.True(t, rate.Rate().IsZero())
	})

	t.Run("rechaza tasa negativa, mayor o igual a 1 o con más de 4 decimales", func(t *testing.T) {
		for _, s := range []string{"-0.01", "1", "19", "0.19001"} {
			_, err := domain.ParseTaxRate(dec(s))
			require.ErrorIs(t, err, domain.ErrInvalidTaxRate, s)
		}
	})
}
