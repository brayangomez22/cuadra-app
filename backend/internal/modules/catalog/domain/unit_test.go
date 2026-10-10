package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestParseUnit(t *testing.T) {
	t.Run("acepta las unidades del catálogo", func(t *testing.T) {
		codes := []string{"und", "m", "kg", "l", "caja", "rollo", "bulto", "galon"}
		for _, code := range codes {
			unit, err := domain.ParseUnit(code)
			require.NoError(t, err)
			require.Equal(t, code, unit.String())
		}
		require.Len(t, domain.Units(), len(codes))
	})

	t.Run("rechaza unidad desconocida", func(t *testing.T) {
		for _, code := range []string{"", "docena", "KG", " m"} {
			_, err := domain.ParseUnit(code)
			require.ErrorIs(t, err, domain.ErrInvalidUnit, code)
		}
	})
}
