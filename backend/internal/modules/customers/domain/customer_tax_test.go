package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

func TestParseTaxResponsibilities(t *testing.T) {
	t.Run("sin responsabilidades queda como no responsable (R-99-PN)", func(t *testing.T) {
		got, err := domain.ParseTaxResponsibilities(nil)
		require.NoError(t, err)
		require.Equal(t, []domain.TaxResponsibility{domain.TaxNotResponsible}, got)
	})

	t.Run("acepta varias responsabilidades y las ordena", func(t *testing.T) {
		got, err := domain.ParseTaxResponsibilities([]string{"O-23", "O-13", "O-15"})
		require.NoError(t, err)
		require.Equal(t, []domain.TaxResponsibility{"O-13", "O-15", "O-23"}, got)
	})

	t.Run("rechaza responsabilidades inválidas", func(t *testing.T) {
		tests := []struct {
			name string
			in   []string
		}{
			{name: "R-99-PN combinada con otra", in: []string{"R-99-PN", "O-13"}},
			{name: "repetida", in: []string{"O-13", "O-13"}},
			{name: "desconocida", in: []string{"O-99"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.ParseTaxResponsibilities(tt.in)
				require.ErrorIs(t, err, domain.ErrInvalidTaxResponsibilities)
			})
		}
	})
}
