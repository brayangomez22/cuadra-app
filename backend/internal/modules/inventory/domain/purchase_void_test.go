package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func purchaseInfo() domain.MovementInfo {
	info := testInfo()
	info.Reference = &domain.DocumentRef{Kind: domain.DocumentPurchase, ID: newID()}
	return info
}

func TestStockLevelReturnPurchase(t *testing.T) {
	t.Run("saca lo que entró al costo de la entrada y deshace su efecto en el costo promedio", func(t *testing.T) {
		// 10 a 1000 y luego 10 a 1200 (la compra que se anula) → promedio 1100.
		level := levelWith(t, "20", "1100")
		info := purchaseInfo()

		m, err := level.ReturnPurchase(dec("10"), dec("1200"), info)
		require.NoError(t, err)

		require.Equal(t, domain.MovementPurchaseVoid, m.Type())
		requireDec(t, "-10", m.Quantity())
		requireDec(t, "1200", m.UnitCost())
		requireDec(t, "10", m.BalanceAfter())
		// (20 × 1100 − 10 × 1200) / 10 = 1000
		requireDec(t, "1000", m.AverageCostAfter())
		require.Equal(t, info.Reference, m.Reference())
		requireDec(t, "10", level.Quantity())
		requireDec(t, "1000", level.AverageCost())
	})

	t.Run("anula metros de un rollo con decimales", func(t *testing.T) {
		// 50 m a 4000 y 25,5 m a 4400 → promedio 4135,0993.
		level := levelWith(t, "75.5", "4135.0993")
		m, err := level.ReturnPurchase(dec("25.5"), dec("4400"), purchaseInfo())
		require.NoError(t, err)
		requireDec(t, "50", m.BalanceAfter())
		// (75,5 × 4135,0993 − 25,5 × 4400) / 50 = 3999,9999: el redondeo del promedio anterior.
		requireDec(t, "3999.9999", level.AverageCost())
	})

	t.Run("si no queda stock el promedio no cambia", func(t *testing.T) {
		level := levelWith(t, "6", "35000")
		m, err := level.ReturnPurchase(dec("6"), dec("35000"), purchaseInfo())
		require.NoError(t, err)
		requireDec(t, "0", m.BalanceAfter())
		requireDec(t, "35000", level.AverageCost())
	})

	t.Run("si deshacer la entrada diera un promedio negativo, el promedio no cambia", func(t *testing.T) {
		// Un pedido mal digitado a 10 veces su costo, con el stock movido después por otras entradas.
		level := levelWith(t, "5", "100")
		m, err := level.ReturnPurchase(dec("4"), dec("1000"), purchaseInfo())
		require.NoError(t, err)
		requireDec(t, "1", m.BalanceAfter())
		requireDec(t, "100", level.AverageCost())
	})

	t.Run("no deja la existencia negativa aunque la empresa lo permita", func(t *testing.T) {
		// Lo que entró ya se vendió en parte: anular la compra no puede inventar stock.
		level := levelWith(t, "3", "1000")
		_, err := level.ReturnPurchase(dec("5"), dec("1000"), purchaseInfo())
		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		requireDec(t, "3", level.Quantity())
	})

	t.Run("purchase_void exige una referencia de compra", func(t *testing.T) {
		for _, info := range []domain.MovementInfo{testInfo(), saleInfo()} {
			level := levelWith(t, "5", "1000")
			_, err := level.ReturnPurchase(dec("1"), dec("1000"), info)
			require.ErrorIs(t, err, domain.ErrInvalidReference)
			requireDec(t, "5", level.Quantity())
		}
	})

	t.Run("rechaza cantidad no positiva o costo negativo", func(t *testing.T) {
		level := levelWith(t, "5", "1000")
		_, err := level.ReturnPurchase(dec("0"), dec("1000"), purchaseInfo())
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
		_, err = level.ReturnPurchase(dec("-1"), dec("1000"), purchaseInfo())
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
		_, err = level.ReturnPurchase(dec("1"), dec("-1"), purchaseInfo())
		require.ErrorIs(t, err, domain.ErrInvalidUnitCost)
	})

	t.Run("su tipo se reconoce al reconstruirlo", func(t *testing.T) {
		mt, err := domain.ParseMovementType("purchase_void")
		require.NoError(t, err)
		require.Equal(t, domain.MovementPurchaseVoid, mt)
	})
}
