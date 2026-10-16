package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func variantInput(sku string, axes map[string]string) app.VariantInput {
	return app.VariantInput{SKU: sku, Cost: dec("15000"), Price: dec("30000"), AxisValues: axes}
}

// productWithAxes creates a product and gives its default variant axes, so
// more variants can be added.
func (e *env) productWithAxes(t *testing.T, axes map[string]string) *domain.Product {
	t.Helper()
	p := e.createProduct(t, e.productInput(t))
	d := p.DefaultVariant()
	in := app.VariantUpdate{
		VariantInput: app.VariantInput{SKU: d.SKU(), Cost: d.Cost(), Price: d.Price(), AxisValues: axes},
		Active:       true,
	}
	p, err := e.svc.UpdateVariant(t.Context(), e.actor, p.ID(), d.ID(), in)
	require.NoError(t, err)
	return p
}

func TestUpdateProductDefaultVariant(t *testing.T) {
	t.Run("actualiza SKU, código y costo de la variante por defecto", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))
		in := e.productInput(t)
		in.SKU, in.Barcode, in.Cost = "tub-pvc-34", "7709876543210", dec("9000")

		updated, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)

		v := updated.DefaultVariant()
		require.Equal(t, p.DefaultVariant().ID(), v.ID())
		require.Equal(t, "TUB-PVC-34", v.SKU())
		require.Equal(t, "7709876543210", v.Barcode())
		require.True(t, dec("9000").Equal(v.Cost()))
	})
}

func TestAddVariant(t *testing.T) {
	tests := []struct {
		name          string
		first, second map[string]string
	}{
		{"almacén de ropa: talla y color", map[string]string{"talla": "S", "color": "Rojo"}, map[string]string{"talla": "M", "color": "Rojo"}},
		{"ferretería: presentación", map[string]string{"presentación": "Galón"}, map[string]string{"presentación": "Cuarto"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("agrega una variante, la guarda y la cuenta", func(t *testing.T) {
				e := newEnv(t)
				p := e.productWithAxes(t, tt.first)

				updated, err := e.svc.AddVariant(t.Context(), e.actor, p.ID(), variantInput("var-2", tt.second))
				require.NoError(t, err)

				require.Len(t, updated.Variants(), 2)
				stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
				require.NoError(t, err)
				require.Len(t, stored.Variants(), 2)
				added := stored.Variants()[1]
				require.Equal(t, "VAR-2", added.SKU())
				require.Equal(t, tt.second, added.AxisValues())
				require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.variants_created", "", ""))

				_, attrs := e.span(t, "catalog.AddVariant")
				require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
				require.Equal(t, p.ID().String(), attrs["product.id"].AsString())
				require.Equal(t, added.ID().String(), attrs["variant.id"].AsString())
			})

			t.Run("rechaza una variante con los mismos ejes y no la guarda", func(t *testing.T) {
				e := newEnv(t)
				p := e.productWithAxes(t, tt.first)

				_, err := e.svc.AddVariant(t.Context(), e.actor, p.ID(), variantInput("VAR-2", tt.first))
				require.ErrorIs(t, err, domain.ErrDuplicateVariant)

				stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
				require.NoError(t, err)
				require.Len(t, stored.Variants(), 1)
				require.Zero(t, e.counter(t, "cuadra.catalog.variants_created", "", ""))
				span, _ := e.span(t, "catalog.AddVariant")
				require.Equal(t, codes.Error, span.Status().Code)
			})
		})
	}

	t.Run("rechaza un SKU que usa otro producto del tenant", func(t *testing.T) {
		e := newEnv(t)
		p := e.productWithAxes(t, map[string]string{"talla": "S"})
		other := e.productInput(t)
		other.SKU = "OTRO-SKU"
		e.createProduct(t, other)

		_, err := e.svc.AddVariant(t.Context(), e.actor, p.ID(), variantInput("otro-sku", map[string]string{"talla": "M"}))
		require.ErrorIs(t, err, domain.ErrSKUTaken)
	})

	t.Run("agregar una variante a un producto de otro tenant da ErrProductNotFound", func(t *testing.T) {
		e := newEnv(t)
		p := e.productWithAxes(t, map[string]string{"talla": "S"})
		other := app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}

		_, err := e.svc.AddVariant(t.Context(), other, p.ID(), variantInput("VAR-2", map[string]string{"talla": "M"}))
		require.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}

func TestUpdateVariant(t *testing.T) {
	t.Run("actualiza una variante, cuenta el cambio de precio y la desactiva", func(t *testing.T) {
		e := newEnv(t)
		p := e.productWithAxes(t, map[string]string{"talla": "S"})
		p, err := e.svc.AddVariant(t.Context(), e.actor, p.ID(), variantInput("CAM-M", map[string]string{"talla": "M"}))
		require.NoError(t, err)
		v := p.Variants()[1]
		in := app.VariantUpdate{
			VariantInput: app.VariantInput{SKU: "cam-m2", Barcode: "7700000000017", Cost: dec("16000"), Price: dec("31000"),
				AxisValues: map[string]string{"talla": "L"}},
			Active: false,
		}

		_, err = e.svc.UpdateVariant(t.Context(), e.actor, p.ID(), v.ID(), in)
		require.NoError(t, err)

		stored, err := e.svc.GetProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		got, err := stored.Variant(v.ID())
		require.NoError(t, err)
		require.Equal(t, "CAM-M2", got.SKU())
		require.Equal(t, "7700000000017", got.Barcode())
		require.True(t, dec("16000").Equal(got.Cost()))
		require.True(t, dec("31000").Equal(got.Price()))
		require.Equal(t, map[string]string{"talla": "L"}, got.AxisValues())
		require.False(t, got.IsActive())
		require.True(t, stored.IsActive())
		// One change for this variant; productWithAxes kept the default's price.
		require.EqualValues(t, 1, e.counter(t, "cuadra.catalog.price_changes", "", ""))

		// productWithAxes ran the first UpdateVariant; this is the second.
		var last map[attribute.Key]attribute.Value
		for _, s := range e.spans.Ended() {
			if s.Name() == "catalog.UpdateVariant" {
				last = map[attribute.Key]attribute.Value{}
				for _, kv := range s.Attributes() {
					last[kv.Key] = kv.Value
				}
			}
		}
		require.Equal(t, p.ID().String(), last["product.id"].AsString())
		require.Equal(t, v.ID().String(), last["variant.id"].AsString())
	})

	t.Run("rechaza cambiar el precio de una variante de un producto inactivo", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))
		_, err := e.svc.DeactivateProduct(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		d := p.DefaultVariant()
		in := app.VariantUpdate{
			VariantInput: app.VariantInput{SKU: d.SKU(), Cost: d.Cost(), Price: dec("99999")},
			Active:       true,
		}

		_, err = e.svc.UpdateVariant(t.Context(), e.actor, p.ID(), d.ID(), in)
		require.ErrorIs(t, err, domain.ErrProductInactive)
	})

	t.Run("una variante de otro producto no se encuentra", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))
		otherIn := e.productInput(t)
		otherIn.SKU = "OTRO"
		other := e.createProduct(t, otherIn)
		d := other.DefaultVariant()
		in := app.VariantUpdate{VariantInput: app.VariantInput{SKU: d.SKU(), Cost: d.Cost(), Price: d.Price()}, Active: true}

		_, err := e.svc.UpdateVariant(t.Context(), e.actor, p.ID(), d.ID(), in)
		require.ErrorIs(t, err, domain.ErrVariantNotFound)
	})
}
