package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestNewPurchase(t *testing.T) {
	t.Run("compra de contado queda sin saldo por pagar", func(t *testing.T) {
		p, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCash),
			[]domain.LineInput{screwBoxes("2", "45000")}, now)
		require.NoError(t, err)
		require.True(t, p.BalanceDue().IsZero())
		require.Nil(t, p.DueOn())
		require.Equal(t, domain.PurchasePaid, p.Status())
	})

	t.Run("compra a crédito queda con saldo igual al total y vence según el plazo del proveedor", func(t *testing.T) {
		p, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCredit),
			[]domain.LineInput{sockDozens("3", "54000")}, now)
		require.NoError(t, err)
		require.True(t, p.BalanceDue().Equal(p.Totals().Total), p.BalanceDue().String())
		require.Equal(t, today.AddDays(28), *p.DueOn())
		require.Equal(t, domain.PurchaseOpen, p.Status())
	})

	t.Run("respeta el vencimiento indicado si no es anterior a la fecha de la factura", func(t *testing.T) {
		params := purchaseParams(t, domain.PaymentCredit)
		due := params.IssuedOn.AddDays(15)
		params.DueOn = &due
		p, err := domain.NewPurchase(tenantID, params, []domain.LineInput{screwBoxes("1", "45000")}, now)
		require.NoError(t, err)
		require.Equal(t, due, *p.DueOn())

		early := params.IssuedOn.AddDays(-1)
		params.DueOn = &early
		_, err = domain.NewPurchase(tenantID, params, []domain.LineInput{screwBoxes("1", "45000")}, now)
		require.ErrorIs(t, err, domain.ErrDueBeforeIssue)
	})

	t.Run("rechaza factura con fecha futura en America/Bogota", func(t *testing.T) {
		params := purchaseParams(t, domain.PaymentCash)
		params.IssuedOn = today.AddDays(1)
		_, err := domain.NewPurchase(tenantID, params, []domain.LineInput{screwBoxes("1", "45000")}, now)
		require.ErrorIs(t, err, domain.ErrInvoiceDateInFuture)
	})

	t.Run("rechaza compra a un proveedor inactivo", func(t *testing.T) {
		params := purchaseParams(t, domain.PaymentCash)
		inactive := supplierParams("Distribuidora del Valle", "900123456-8")
		inactive.Active = false
		require.NoError(t, params.Supplier.Update(inactive))
		_, err := domain.NewPurchase(tenantID, params, []domain.LineInput{screwBoxes("1", "45000")}, now)
		require.ErrorIs(t, err, domain.ErrSupplierInactive)
	})

	t.Run("rechaza compra sin líneas o con un medio de pago desconocido", func(t *testing.T) {
		_, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCash), nil, now)
		require.ErrorIs(t, err, domain.ErrNoLines)
		_, err = domain.NewPurchase(tenantID, purchaseParams(t, domain.Payment("check")),
			[]domain.LineInput{screwBoxes("1", "45000")}, now)
		require.ErrorIs(t, err, domain.ErrInvalidPayment)
	})

	t.Run("rechaza líneas con cantidad, costo o tasa inválidos", func(t *testing.T) {
		badRate := sockDozens("1", "54000")
		badRate.TaxRate = dec("1.5")
		for _, tc := range []struct {
			name string
			line domain.LineInput
			want error
		}{
			{"cantidad cero", screwBoxes("0", "45000"), domain.ErrInvalidQuantity},
			{"cantidad con más de 4 decimales", sockDozens("1.00001", "54000"), domain.ErrInvalidQuantity},
			{"costo negativo", screwBoxes("1", "-1"), domain.ErrInvalidUnitCost},
			{"tasa mayor o igual a 1", badRate, domain.ErrInvalidTaxRate},
		} {
			_, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCash),
				[]domain.LineInput{tc.line}, now)
			require.ErrorIs(t, err, tc.want, tc.name)
		}
	})

	t.Run("una bonificación del proveedor entra con costo cero", func(t *testing.T) {
		p, err := domain.NewPurchase(tenantID, purchaseParams(t, domain.PaymentCash),
			[]domain.LineInput{sockDozens("1", "0")}, now)
		require.NoError(t, err)
		require.True(t, p.Totals().Total.IsZero())
	})
}

func TestInvoiceNumber(t *testing.T) {
	t.Run("normaliza el número de factura sin espacios ni mayúsculas", func(t *testing.T) {
		for _, in := range []string{"FE 1234", " fe1234 ", "Fe-1234"} {
			params := purchaseParams(t, domain.PaymentCash)
			params.InvoiceNumber = in
			p, err := domain.NewPurchase(tenantID, params, []domain.LineInput{screwBoxes("1", "1")}, now)
			require.NoError(t, err, in)
			require.Equal(t, "FE1234", p.InvoiceKey(), in)
		}
	})

	t.Run("rechaza número de factura vacío o muy largo", func(t *testing.T) {
		for _, in := range []string{"  ", string(make([]byte, 51))} {
			params := purchaseParams(t, domain.PaymentCash)
			params.InvoiceNumber = in
			_, err := domain.NewPurchase(tenantID, params, []domain.LineInput{screwBoxes("1", "1")}, now)
			require.ErrorIs(t, err, domain.ErrInvalidInvoiceNumber)
		}
	})
}
