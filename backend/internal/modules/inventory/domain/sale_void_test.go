package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func saleInfo() domain.MovementInfo {
	info := testInfo()
	info.Reference = &domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()}
	return info
}

func TestStockLevelReturnSale(t *testing.T) {
	t.Run("la anulación de una venta suma la cantidad y recalcula el costo promedio con el costo original", func(t *testing.T) {
		// 10 a 1000; luego entraron 10 a 1200 → promedio 1100. La venta anulada salió a 1000.
		level := levelWith(t, "20", "1100")
		info := saleInfo()

		m, err := level.ReturnSale(dec("4"), dec("1000"), info)
		require.NoError(t, err)

		require.Equal(t, domain.MovementSaleVoid, m.Type())
		requireDec(t, "4", m.Quantity())
		requireDec(t, "1000", m.UnitCost())
		requireDec(t, "24", m.BalanceAfter())
		// (20 × 1100 + 4 × 1000) / 24 = 1083.3333
		requireDec(t, "1083.3333", m.AverageCostAfter())
		require.Equal(t, info.Reference, m.Reference())
		requireDec(t, "24", level.Quantity())
		requireDec(t, "1083.3333", level.AverageCost())
	})

	t.Run("con saldo cero el promedio pasa a ser el costo original", func(t *testing.T) {
		level := levelWith(t, "0", "900")
		m, err := level.ReturnSale(dec("2.5"), dec("1000"), saleInfo())
		require.NoError(t, err)
		requireDec(t, "2.5", m.BalanceAfter())
		requireDec(t, "1000", level.AverageCost())
	})

	t.Run("sale_void exige una referencia de venta", func(t *testing.T) {
		purchase := testInfo()
		purchase.Reference = &domain.DocumentRef{Kind: domain.DocumentPurchase, ID: newID()}
		for _, info := range []domain.MovementInfo{testInfo(), purchase} {
			level := levelWith(t, "5", "1000")
			_, err := level.ReturnSale(dec("1"), dec("1000"), info)
			require.ErrorIs(t, err, domain.ErrInvalidReference)
			requireDec(t, "5", level.Quantity())
		}
	})

	t.Run("rechaza cantidad no positiva o costo negativo", func(t *testing.T) {
		level := levelWith(t, "5", "1000")
		_, err := level.ReturnSale(dec("0"), dec("1000"), saleInfo())
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
		_, err = level.ReturnSale(dec("-1"), dec("1000"), saleInfo())
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
		_, err = level.ReturnSale(dec("1"), dec("-1"), saleInfo())
		require.ErrorIs(t, err, domain.ErrInvalidUnitCost)
	})

	t.Run("reconstruye un sale_void guardado y su tipo se reconoce", func(t *testing.T) {
		mt, err := domain.ParseMovementType("sale_void")
		require.NoError(t, err)
		require.Equal(t, domain.MovementSaleVoid, mt)
	})
}
