package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestPurchaseTotals(t *testing.T) {
	t.Run("calcula subtotal, IVA por tasa y total con redondeo half-up", func(t *testing.T) {
		exempt := sockDozens("2.5", "10001.10")
		exempt.TaxRate = dec("0")
		p, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCredit), []domain.LineInput{
			screwBoxes("3", "12345.6789"), // 37037.0367 → 37037.04; IVA 7037.0376 → 7037.04
			sockDozens("1", "100.05"),     // 100.05; IVA 19.0095 → 19.01
			exempt,                        // 25002.75; sin IVA
		}, now)
		require.NoError(t, err)

		lines := p.Lines()
		require.Equal(t, "37037.04", lines[0].Subtotal().StringFixed(2))
		require.Equal(t, "7037.04", lines[0].Tax().StringFixed(2))

		totals := p.Totals()
		require.Equal(t, "62139.84", totals.Subtotal.StringFixed(2))
		require.Equal(t, "7056.05", totals.Tax.StringFixed(2))
		require.Equal(t, "69195.89", totals.Total.StringFixed(2))
		require.Len(t, totals.Taxes, 2)
		require.Equal(t, "0", totals.Taxes[0].Rate.String())
		require.Equal(t, "25002.75", totals.Taxes[0].Base.StringFixed(2))
		require.Equal(t, "0.19", totals.Taxes[1].Rate.String())
		require.Equal(t, "37137.09", totals.Taxes[1].Base.StringFixed(2))
		require.Equal(t, "7056.05", totals.Taxes[1].Tax.StringFixed(2))
	})
}

func TestPurchaseReceive(t *testing.T) {
	t.Run("la línea guarda la unidad de compra, su cantidad y su costo, y lo que entró en unidad base", func(t *testing.T) {
		p, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCash),
			[]domain.LineInput{screwBoxes("2", "45000"), sockDozens("3", "54000")}, now)
		require.NoError(t, err)
		require.NoError(t, p.Receive([]domain.Received{
			{BaseQuantity: dec("200"), BaseUnitCost: dec("450")},
			{BaseQuantity: dec("36"), BaseUnitCost: dec("4500")},
		}))

		boxes := p.Lines()[0]
		require.Equal(t, "caja", boxes.Unit().Code)
		require.Equal(t, "2", boxes.Quantity().String())
		require.Equal(t, "45000", boxes.UnitCost().String())
		require.Equal(t, "200", boxes.BaseQuantity().String())
		require.Equal(t, "450", boxes.BaseUnitCost().String())
	})

	t.Run("rechaza un recibo que no trae una entrada por línea", func(t *testing.T) {
		p, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCash),
			[]domain.LineInput{screwBoxes("2", "45000")}, now)
		require.NoError(t, err)
		require.ErrorIs(t, p.Receive(nil), domain.ErrReceiptMismatch)
	})

	t.Run("se reconstruye igual desde lo guardado", func(t *testing.T) {
		p, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCredit),
			[]domain.LineInput{sockDozens("3", "54000")}, now)
		require.NoError(t, err)
		require.NoError(t, p.Receive([]domain.Received{{BaseQuantity: dec("36"), BaseUnitCost: dec("4500")}}))
		back, err := domain.RehydratePurchase(p.Snapshot())
		require.NoError(t, err)
		require.Equal(t, p.Snapshot(), back.Snapshot())
	})
}
