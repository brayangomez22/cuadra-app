package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
)

func TestAuditOfPriceChanges(t *testing.T) {
	t.Run("ferretería: editar el producto con otro precio deja un evento con el anterior y el nuevo", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t)) // price 10000
		in := e.productInput(t)
		in.Price = dec("10800")

		_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)

		events := e.audit.of("variant.price_changed")
		require.Len(t, events, 1)
		require.Equal(t, p.DefaultVariant().ID(), events[0].EntityID)
		require.Equal(t, map[string]string{"old_price": "10000.00", "new_price": "10800.00"}, events[0].Details)
	})

	t.Run("almacén de ropa: editar una talla con otro precio deja un evento", func(t *testing.T) {
		e := newEnv(t)
		p := e.productWithAxes(t, map[string]string{"talla": "M"})
		v := p.DefaultVariant()
		in := app.VariantUpdate{
			VariantInput: app.VariantInput{SKU: v.SKU(), Cost: v.Cost(), Price: dec("45900"), AxisValues: map[string]string{"talla": "M"}},
			Active:       true,
		}

		_, err := e.svc.UpdateVariant(t.Context(), e.actor, p.ID(), v.ID(), in)
		require.NoError(t, err)

		events := e.audit.of("variant.price_changed")
		require.Len(t, events, 1)
		require.Equal(t, v.ID(), events[0].EntityID)
		require.Equal(t, map[string]string{"old_price": "10000.00", "new_price": "45900.00"}, events[0].Details)
	})

	t.Run("editar sin cambiar el precio no deja evento", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))
		in := e.productInput(t)
		in.Name = "Tubo PVC 1/2"

		_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.NoError(t, err)
		require.Empty(t, e.audit.entries)
	})

	t.Run("una importación que cambia un precio deja su evento", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))
		v := app.VariantInput{SKU: "tub-pvc-12", Cost: dec("8000"), Price: dec("11500")}

		_, err := e.svc.ImportProductDrafts(t.Context(), e.actor, []app.ProductDraft{draft("Tubo", "", v)}, app.UpdateExisting)
		require.NoError(t, err)

		events := e.audit.of("variant.price_changed")
		require.Len(t, events, 1)
		require.Equal(t, p.DefaultVariant().ID(), events[0].EntityID)
		require.Equal(t, map[string]string{"old_price": "10000.00", "new_price": "11500.00"}, events[0].Details)
	})

	t.Run("si el registro falla, el precio no cambia", func(t *testing.T) {
		e := newEnv(t)
		p := e.createProduct(t, e.productInput(t))
		e.audit.fail = errors.New("audit down")
		in := e.productInput(t)
		in.Price = dec("10800")

		_, err := e.svc.UpdateProduct(t.Context(), e.actor, p.ID(), in)
		require.ErrorContains(t, err, "audit down")
	})
}
