package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestSetVariantUnit(t *testing.T) {
	tests := []struct {
		name           string
		code, unitName string
		factor, price  string
	}{
		{"ferretería: rollo de 100 m de un producto que se vende por metro", "rollo", "Rollo", "100", "950000"},
		{"almacén de ropa: paquete de 3 de un producto que se vende por unidad", "paquete", "Paquete", "3", "28000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("agrega la unidad alternativa con el código y el nombre de la unidad del tenant", func(t *testing.T) {
				e := newEnv(t)
				unit := e.addUnit(t, e.actor.TenantID, tt.code, tt.unitName)
				p := e.createProduct(t, e.productInput(t))
				v := p.DefaultVariant()

				got, err := e.svc.SetVariantUnit(t.Context(), e.actor, p.ID(), v.ID(), unit.ID(), app.VariantUnitInput{
					Factor: dec(tt.factor), Price: dec(tt.price), Barcode: "7705555555555",
				})
				require.NoError(t, err)

				units := got.DefaultVariant().Units()
				require.Len(t, units, 1)
				require.Equal(t, domain.UnitRef{ID: unit.ID(), Code: tt.code, Name: tt.unitName}, units[0].Unit())
				require.True(t, dec(tt.factor).Equal(units[0].Factor()))
				require.True(t, dec(tt.price).Equal(units[0].Price()))
				require.Equal(t, "7705555555555", units[0].Barcode())

				stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
				require.NoError(t, err)
				require.Len(t, stored.DefaultVariant().Units(), 1)

				_, attrs := e.span(t, "catalog.SetVariantUnit")
				require.Equal(t, p.ID().String(), attrs["product.id"].AsString())
				require.Equal(t, v.ID().String(), attrs["variant.id"].AsString())
				require.Equal(t, unit.ID().String(), attrs["unit.id"].AsString())
			})

			t.Run("quita la unidad alternativa", func(t *testing.T) {
				e := newEnv(t)
				unit := e.addUnit(t, e.actor.TenantID, tt.code, tt.unitName)
				p := e.createProduct(t, e.productInput(t))
				v := p.DefaultVariant()
				_, err := e.svc.SetVariantUnit(t.Context(), e.actor, p.ID(), v.ID(), unit.ID(), app.VariantUnitInput{
					Factor: dec(tt.factor), Price: dec(tt.price),
				})
				require.NoError(t, err)

				got, err := e.svc.RemoveVariantUnit(t.Context(), e.actor, p.ID(), v.ID(), unit.ID())
				require.NoError(t, err)
				require.Empty(t, got.DefaultVariant().Units())

				stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
				require.NoError(t, err)
				require.Empty(t, stored.DefaultVariant().Units())
				e.span(t, "catalog.RemoveVariantUnit")
			})
		})
	}

	t.Run("rechaza una unidad que no existe en el tenant", func(t *testing.T) {
		e := newEnv(t)
		foreign := e.addUnit(t, uuid.Must(uuid.NewV7()), "caja", "Caja")
		p := e.createProduct(t, e.productInput(t))

		_, err := e.svc.SetVariantUnit(t.Context(), e.actor, p.ID(), p.DefaultVariant().ID(), foreign.ID(), app.VariantUnitInput{
			Factor: dec("12"), Price: dec("100000"),
		})
		require.ErrorIs(t, err, domain.ErrUnitNotFound)
	})

	t.Run("rechaza un producto de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		unit := e.addUnit(t, e.actor.TenantID, "caja", "Caja")
		p := e.createProduct(t, e.productInput(t))
		other := app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}

		_, err := e.svc.SetVariantUnit(t.Context(), other, p.ID(), p.DefaultVariant().ID(), unit.ID(), app.VariantUnitInput{
			Factor: dec("12"), Price: dec("100000"),
		})
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("rechaza el código de barras de una variante de otro producto", func(t *testing.T) {
		e := newEnv(t)
		unit := e.addUnit(t, e.actor.TenantID, "caja", "Caja")
		in := e.productInput(t)
		in.Barcode = "7706666666666"
		e.createProduct(t, in)
		second := e.productInput(t)
		second.SKU = "OTRO-1"
		p := e.createProduct(t, second)

		_, err := e.svc.SetVariantUnit(t.Context(), e.actor, p.ID(), p.DefaultVariant().ID(), unit.ID(), app.VariantUnitInput{
			Factor: dec("12"), Price: dec("100000"), Barcode: "7706666666666",
		})
		require.ErrorIs(t, err, domain.ErrBarcodeTaken)
	})

	t.Run("rechaza crear un producto con el código de barras de una unidad alternativa de otro", func(t *testing.T) {
		e := newEnv(t)
		unit := e.addUnit(t, e.actor.TenantID, "caja", "Caja")
		p := e.createProduct(t, e.productInput(t))
		_, err := e.svc.SetVariantUnit(t.Context(), e.actor, p.ID(), p.DefaultVariant().ID(), unit.ID(), app.VariantUnitInput{
			Factor: dec("12"), Price: dec("100000"), Barcode: "7707777777777",
		})
		require.NoError(t, err)

		second := e.productInput(t)
		second.SKU, second.Barcode = "OTRO-1", "7707777777777"
		_, err = e.svc.CreateProduct(t.Context(), e.actor, second)
		require.ErrorIs(t, err, domain.ErrBarcodeTaken)
	})

	t.Run("rechaza quitar una unidad que la variante no tiene", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))

		_, err := e.svc.RemoveVariantUnit(t.Context(), e.actor, p.ID(), p.DefaultVariant().ID(), uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrVariantUnitNotFound)
	})
}
