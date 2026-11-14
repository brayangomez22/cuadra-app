//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSaleLineCodes(t *testing.T) {
	f := newFixture(t)

	t.Run("la venta guarda el SKU, la unidad base, su código DIAN y el tratamiento del IVA de cada línea", func(t *testing.T) {
		tn := f.createTenant(t)
		s, err := domain.NewSale(domain.NewSaleParams{ID: newID(), TenantID: tn.id, LocationID: tn.location, UserID: newID(), CreatedAt: now})
		require.NoError(t, err)
		require.NoError(t, s.AddLine(domain.LineInput{
			ProductID: tn.product, VariantID: tn.product, ProductName: "Cemento gris 50kg", Quantity: dec("2"),
			UnitPrice: dec("30000"), TaxRate: dec("0.19"), Discount: dec("0"), PriceRule: domain.VariantPriceRule,
			SKU: "CEM-50", BaseUnitCode: "und", UnitDIANCode: "94", TaxTreatment: vat.Taxed,
		}, domain.NoDiscountLimit))
		require.NoError(t, s.AddLine(domain.LineInput{
			ProductID: tn.product, VariantID: tn.variant, ProductName: "Cemento gris 50kg", VariantLabel: "25kg",
			Quantity: dec("1"), UnitPrice: dec("15000"), TaxRate: dec("0"), Discount: dec("0"),
			PriceRule: domain.VariantPriceRule, SKU: "CEM-25", BaseUnitCode: "und", TaxTreatment: vat.Exempt,
		}, domain.NoDiscountLimit))
		cash, err := domain.NewPayment(domain.PaymentCash, dec("86400"))
		require.NoError(t, err)
		require.NoError(t, s.Confirm([]domain.Payment{cash}, now))

		f.save(t, s)

		got, err := f.sales.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		lines := got.Lines()
		require.Equal(t, "CEM-50", lines[0].SKU())
		require.Equal(t, "und", lines[0].BaseUnitCode())
		require.Equal(t, "94", lines[0].UnitDIANCode())
		require.Equal(t, vat.Taxed, lines[0].TaxTreatment())
		require.Equal(t, "CEM-25", lines[1].SKU())
		require.Empty(t, lines[1].UnitDIANCode())
		require.Equal(t, vat.Exempt, lines[1].TaxTreatment())
	})

	t.Run("aislamiento: las líneas del tenant A no se ven desde el B", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		s := confirmed(t, a, a.location, now)
		f.save(t, s)

		_, err := f.sales.Get(t.Context(), b.id, s.ID())
		require.ErrorIs(t, err, domain.ErrSaleNotFound)
	})
}
