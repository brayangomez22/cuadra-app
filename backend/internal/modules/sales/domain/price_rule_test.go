package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestNewPriceRuleID(t *testing.T) {
	t.Run("acepta ids estables en snake_case", func(t *testing.T) {
		for _, in := range []string{"variant_price", "price_lists", "volume_pricing", "promo2"} {
			id, err := domain.NewPriceRuleID(in)
			require.NoError(t, err, in)
			require.Equal(t, domain.PriceRuleID(in), id)
		}
	})

	t.Run("rechaza ids vacíos, con mayúsculas, espacios o de más de 64 caracteres", func(t *testing.T) {
		for _, in := range []string{"", "Price_lists", "lista mayorista", "2x1", "_promo", "precio-lista", strings.Repeat("a", 65)} {
			_, err := domain.NewPriceRuleID(in)
			require.ErrorIs(t, err, domain.ErrInvalidPriceRule, in)
		}
	})

	t.Run("la regla base es variant_price", func(t *testing.T) {
		require.Equal(t, domain.VariantPriceRule, domain.PriceRuleID("variant_price"))
	})
}
