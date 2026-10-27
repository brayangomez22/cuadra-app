package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestUpdateProductBaseUnit(t *testing.T) {
	tests := []struct {
		business string
		code     string
	}{
		{"ferretería: de metro a kilogramo", "kg"},
		{"almacén de ropa: de metro a par", "par"},
	}
	for _, tt := range tests {
		t.Run(tt.business, func(t *testing.T) {
			t.Run("rechaza cambiar la unidad base si el producto tiene movimientos de stock", func(t *testing.T) {
				e := newEnv(t)
				p := e.createProduct(t, e.productInput(t))
				e.store.moved[p.DefaultVariant().ID()] = true
				in := e.productInput(t)
				in.BaseUnitID = e.addUnit(t, e.actor.TenantID, tt.code, "Otra").ID()

				_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
				require.ErrorIs(t, err, domain.ErrBaseUnitLocked)
			})

			t.Run("sin movimientos cambia la unidad base", func(t *testing.T) {
				e := newEnv(t)
				p := e.createProduct(t, e.productInput(t))
				in := e.productInput(t)
				in.BaseUnitID = e.addUnit(t, e.actor.TenantID, tt.code, "Otra").ID()

				updated, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
				require.NoError(t, err)
				require.Equal(t, in.BaseUnitID, updated.BaseUnit().ID)
			})

			t.Run("con movimientos acepta los demás cambios", func(t *testing.T) {
				e := newEnv(t)
				p := e.createProduct(t, e.productInput(t))
				e.store.moved[p.DefaultVariant().ID()] = true
				in := e.productInput(t)
				in.Name = "Otro nombre"

				_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
				require.NoError(t, err)
			})
		})
	}
}
