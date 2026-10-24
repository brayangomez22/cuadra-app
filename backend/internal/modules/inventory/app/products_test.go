package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
)

func TestProductOfEachVariant(t *testing.T) {
	t.Run("las entradas, ajustes y traslados devuelven el producto de su variante", func(t *testing.T) {
		e := newEnv(t)
		productID, variants := e.addProduct(1)
		from, to := e.createLocation(t, "Principal"), e.createLocation(t, "Bodega norte")

		received, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			VariantID: variants[1], LocationID: from.ID(), Quantity: dec("10"), UnitCost: dec("1500"),
		})
		require.NoError(t, err)
		require.Equal(t, productID, received.ProductID)

		adjusted, err := e.svc.AdjustStock(t.Context(), e.actor, app.AdjustmentInput{
			VariantID: variants[1], LocationID: from.ID(), Quantity: dec("-1"), Reason: "Conteo",
		})
		require.NoError(t, err)
		require.Equal(t, productID, adjusted.ProductID)

		out, in, err := e.svc.TransferStock(t.Context(), e.actor, app.TransferInput{
			VariantID: variants[1], FromLocationID: from.ID(), ToLocationID: to.ID(), Quantity: dec("2"),
		})
		require.NoError(t, err)
		require.Equal(t, productID, out.ProductID)
		require.Equal(t, productID, in.ProductID)
	})

	t.Run("las existencias y el kardex traen el producto de cada variante", func(t *testing.T) {
		e := newEnv(t)
		productID, variants := e.addProduct(1)
		loc := e.createLocation(t, "Principal")
		e.receive(t, variants[0], loc.ID(), "3", "1000")
		e.receive(t, variants[1], loc.ID(), "5", "2000")

		stock, err := e.svc.ListStock(t.Context(), e.actor, app.StockQuery{LocationID: loc.ID()})
		require.NoError(t, err)
		require.Len(t, stock.Items, 2)
		for _, l := range stock.Items {
			require.Equal(t, productID, l.ProductID)
		}

		kardex, err := e.svc.GetKardex(t.Context(), e.actor, app.KardexQuery{VariantID: variants[1]})
		require.NoError(t, err)
		require.Len(t, kardex.Items, 1)
		require.Equal(t, productID, kardex.Items[0].ProductID)
	})

	t.Run("una variante que el catálogo no conoce queda sin producto", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Principal")

		m, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
			VariantID: newID(), LocationID: loc.ID(), Quantity: dec("1"), UnitCost: dec("1000"),
		})
		require.NoError(t, err)
		require.Zero(t, m.ProductID)
	})
}
