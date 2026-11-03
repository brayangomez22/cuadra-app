package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestReversePurchaseStock(t *testing.T) {
	buy := func(t *testing.T, e *env, locationID, purchaseID uuid.UUID, lines ...app.PurchaseStockLine) {
		t.Helper()
		_, err := e.svc.ReceivePurchaseStock(t.Context(), e.actor, app.PurchaseStockInput{
			LocationID: locationID, PurchaseID: purchaseID, Lines: lines,
		})
		require.NoError(t, err)
	}

	t.Run("saca exactamente lo que entró con la compra, al costo de su entrada", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		screws, socks := newID(), newID()
		e.receive(t, screws, loc.ID(), "100", "400")
		purchaseID := newID()
		// Ferretería: 2 cajas de 100; almacén de ropa: 3 docenas.
		buy(t, e, loc.ID(), purchaseID, purchaseLine(screws, "2", "45000", "100"), purchaseLine(socks, "3", "54000", "12"))

		movements, err := e.svc.ReversePurchaseStock(t.Context(), e.actor, purchaseID)
		require.NoError(t, err)

		require.Len(t, movements, 2)
		for _, m := range movements {
			require.Equal(t, domain.MovementPurchaseVoid, m.Type())
			require.Equal(t, &domain.DocumentRef{Kind: domain.DocumentPurchase, ID: purchaseID}, m.Reference())
			require.Equal(t, loc.ID(), m.LocationID())
		}
		requireDecimal(t, "100", e.level(screws, loc.ID()).Quantity)
		// (300 × 433,3333 − 200 × 450) / 100: vuelve al costo anterior salvo el redondeo del promedio.
		requireDecimal(t, "399.9999", e.level(screws, loc.ID()).AverageCost)
		requireDecimal(t, "0", e.level(socks, loc.ID()).Quantity)

		require.Equal(t, int64(2), e.counter(t, "cuadra.inventory.movements", "type", "purchase_void"))
		_, attrs := e.span(t, "inventory.ReversePurchaseStock")
		require.Equal(t, purchaseID.String(), attrs["purchase.id"].AsString())
	})

	t.Run("rechaza si ya se vendió parte y nombra la variante; no saca nada", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		paint, shoes := newID(), newID()
		purchaseID := newID()
		buy(t, e, loc.ID(), purchaseID, purchaseLine(paint, "4", "98000", "1"), purchaseLine(shoes, "2", "120000", "1"))
		_, err := e.svc.IssueSaleStock(t.Context(), e.actor, app.SaleStockInput{
			LocationID: loc.ID(), SaleID: newID(), Lines: []app.SaleStockLine{saleLine(shoes, "1")},
		})
		require.NoError(t, err)

		_, err = e.svc.ReversePurchaseStock(t.Context(), e.actor, purchaseID)

		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		var variantErr *app.VariantStockError
		require.ErrorAs(t, err, &variantErr)
		require.Equal(t, shoes, variantErr.VariantID)
		require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.insufficient_stock", "type", "purchase_void"))
	})

	t.Run("no saca dos veces el stock de la misma compra", func(t *testing.T) {
		e := newEnv(t)
		loc := e.createLocation(t, "Bodega")
		p := newID()
		purchaseID := newID()
		buy(t, e, loc.ID(), purchaseID, purchaseLine(p, "10", "1000", "1"))
		_, err := e.svc.ReversePurchaseStock(t.Context(), e.actor, purchaseID)
		require.NoError(t, err)

		_, err = e.svc.ReversePurchaseStock(t.Context(), e.actor, purchaseID)
		require.ErrorIs(t, err, domain.ErrPurchaseAlreadyReversed)
		requireDecimal(t, "0", e.level(p, loc.ID()).Quantity)
	})

	t.Run("una compra sin entradas no saca nada", func(t *testing.T) {
		e := newEnv(t)
		movements, err := e.svc.ReversePurchaseStock(t.Context(), e.actor, newID())
		require.NoError(t, err)
		require.Empty(t, movements)
	})
}
