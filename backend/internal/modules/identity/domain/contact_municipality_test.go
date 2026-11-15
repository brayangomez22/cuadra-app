package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestContactWithMunicipality(t *testing.T) {
	base, err := domain.NewContact("Cra 50 # 40-20", "Medellín", "")
	require.NoError(t, err)

	cases := []struct {
		name string
		code string
		err  error
	}{
		{"ferretería de Medellín con su código DANE", "05001", nil},
		{"almacén de ropa de Bogotá con su código DANE", "11001", nil},
		{"vacío borra el municipio", "", nil},
		{"rechaza un código de cuatro dígitos", "5001", domain.ErrInvalidMunicipality},
		{"rechaza un código con letras", "05OO1", domain.ErrInvalidMunicipality},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := base.WithMunicipality(tc.code)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.code, got.MunicipalityCode())
			require.Equal(t, base.City(), got.City())
		})
	}
}
