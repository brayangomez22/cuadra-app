package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestProductBaseUnitChange(t *testing.T) {
	tests := []struct {
		name      string
		base      domain.UnitRef
		alternate domain.VariantUnitParams
		other     domain.UnitRef
	}{
		{
			name:      "ferretería: un cable por metro con rollo de 100 m",
			base:      meters,
			alternate: domain.VariantUnitParams{Unit: roll, Factor: dec("100"), Price: dec("180000")},
			other:     kilos,
		},
		{
			name:      "almacén de ropa: camisetas por unidad con paquete de 3",
			base:      piece,
			alternate: domain.VariantUnitParams{Unit: pack, Factor: dec("3"), Price: dec("75000")},
			other:     dozens,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("rechaza cambiar la unidad base si una variante tiene unidades alternativas", func(t *testing.T) {
				product := productIn(t, tt.base, true)
				require.NoError(t, product.SetVariantUnit(product.Variants()[1].ID(), tt.alternate))
				d := validDetails(t)
				d.BaseUnit = tt.other

				require.ErrorIs(t, product.UpdateDetails(d), domain.ErrBaseUnitLocked)
				require.Equal(t, tt.base, product.BaseUnit())
			})

			t.Run("con unidades alternativas acepta los demás cambios si la unidad base sigue igual", func(t *testing.T) {
				product := productIn(t, tt.base, true)
				require.NoError(t, product.SetVariantUnit(product.Variants()[1].ID(), tt.alternate))
				d := validDetails(t)
				d.BaseUnit = tt.base

				require.NoError(t, product.UpdateDetails(d))
			})

			t.Run("sin unidades alternativas cambia la unidad base", func(t *testing.T) {
				product := productIn(t, tt.base, true)
				d := validDetails(t)
				d.BaseUnit = tt.other

				require.NoError(t, product.UpdateDetails(d))
				require.Equal(t, tt.other, product.BaseUnit())
			})
		})
	}
}
