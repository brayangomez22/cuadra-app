package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
)

func TestValidateSteps(t *testing.T) {
	cases := []struct {
		name  string
		steps []string
		err   error
	}{
		{"acepta los pasos de una ferretería", hardwareSteps, nil},
		{"acepta los pasos de un almacén de ropa", clothingSteps, nil},
		{"rechaza una plantilla sin pasos", nil, domain.ErrNoSteps},
		{"rechaza un paso desconocido en una plantilla", []string{"products", "pour_concrete"}, domain.ErrUnknownStep},
		{"rechaza un paso repetido", []string{"products", "first_sale", "products"}, domain.ErrRepeatedStep},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.ValidateSteps(tc.steps)
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestStepCatalog(t *testing.T) {
	t.Run("cada paso del catálogo dice qué permiso lo resuelve", func(t *testing.T) {
		for _, s := range domain.Steps() {
			require.NotEmpty(t, s.Permission, s.ID)
		}
	})
}
