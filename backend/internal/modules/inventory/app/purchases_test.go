package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func purchaseLine(variantID uuid.UUID, qty, cost, factor string) app.PurchaseStockLine {
	return app.PurchaseStockLine{VariantID: variantID, Quantity: dec(qty), UnitCost: dec(cost), Factor: dec(factor)}
}

func TestReceivePurchaseStock(t *testing.T) {
	t.Run("registra una entrada por línea con el costo convertido a la unidad base y en el orden de las líneas", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		screws, socks := newID(), newID()
		e.receive(t, screws, loc.ID(), "100", "400")
		purchaseID := newID()

		// Ferretería: 2 cajas de 100 a 45.000; almacén de ropa: 3 docenas a 54.000.
		movements, err := e.svc.ReceivePurchaseStock(t.Context(), e.actor, app.PurchaseStockInput{
			LocationID: loc.ID(), PurchaseID: purchaseID,
			Lines: []app.PurchaseStockLine{purchaseLine(socks, "3", "54000", "12"), purchaseLine(screws, "2", "45000", "100")},
		})
		require.NoError(t, err)

		require.Len(t, movements, 2)
		require.Equal(t, socks, movements[0].VariantID())
		requireDecimal(t, "36", movements[0].Quantity())
		requireDecimal(t, "4500", movements[0].UnitCost())
		require.Equal(t, screws, movements[1].VariantID())
		requireDecimal(t, "200", movements[1].Quantity())
		requireDecimal(t, "450", movements[1].UnitCost())
		for _, m := range movements {
			require.Equal(t, domain.MovementPurchaseIn, m.Type())
			require.Equal(t, &domain.DocumentRef{Kind: domain.DocumentPurchase, ID: purchaseID}, m.Reference())
		}
		// (100 × 400 + 200 × 450) / 300
		requireDecimal(t, "433.3333", e.level(screws, loc.ID()).AverageCost)
		requireDecimal(t, "300", e.level(screws, loc.ID()).Quantity)

		require.Equal(t, int64(3), e.counter(t, "cuadra.inventory.movements", "type", "purchase_in"))
		_, attrs := e.span(t, "inventory.ReceivePurchaseStock")
		require.Equal(t, purchaseID.String(), attrs["purchase.id"].AsString())
	})

	t.Run("dos líneas de la misma variante en unidades distintas suman las dos", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		v := newID()
		_, err := e.svc.ReceivePurchaseStock(t.Context(), e.actor, app.PurchaseStockInput{
			LocationID: loc.ID(), PurchaseID: newID(),
			Lines: []app.PurchaseStockLine{purchaseLine(v, "1", "45000", "100"), purchaseLine(v, "50", "450", "1")},
		})
		require.NoError(t, err)
		requireDecimal(t, "150", e.level(v, loc.ID()).Quantity)
	})

	t.Run("rechaza una cantidad que en unidad base necesita más de 4 decimales y dice la variante", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		v := newID()
		_, err := e.svc.ReceivePurchaseStock(t.Context(), e.actor, app.PurchaseStockInput{
			LocationID: loc.ID(), PurchaseID: newID(), Lines: []app.PurchaseStockLine{purchaseLine(v, "0.0001", "1", "0.5")},
		})
		require.ErrorIs(t, err, domain.ErrQuantityPrecision)
		var variantErr *app.VariantStockError
		require.ErrorAs(t, err, &variantErr)
		require.Equal(t, v, variantErr.VariantID)
	})

	t.Run("rechaza la entrada en una sede inactiva", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		e.deactivate(loc.ID())
		_, err := e.svc.ReceivePurchaseStock(t.Context(), e.actor, app.PurchaseStockInput{
			LocationID: loc.ID(), PurchaseID: newID(), Lines: []app.PurchaseStockLine{purchaseLine(newID(), "1", "1", "1")},
		})
		require.ErrorIs(t, err, domain.ErrLocationInactive)
	})
}
