//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// withTax gives the product a rate and its IVA treatment.
func withTax(t *testing.T, rate string, treatment vat.Treatment) productOption {
	t.Helper()
	r, err := domain.NewTaxRate(dec(rate), treatment)
	require.NoError(t, err)
	return func(p *domain.NewProductParams) { p.TaxRate = r }
}

func TestElectronicDocumentData(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda, edita y lee el código DIAN de la unidad", func(t *testing.T) {
		tenant := f.createTenant(t)
		u := f.createUnit(t, tenant, "m", "Metro")
		require.NoError(t, u.SetDIANCode("MTR"))
		require.NoError(t, f.units.Update(t.Context(), u))

		got, err := f.units.GetByID(t.Context(), tenant, u.ID())
		require.NoError(t, err)
		require.Equal(t, "MTR", got.DIANCode())

		require.NoError(t, u.SetDIANCode(""))
		require.NoError(t, f.units.Update(t.Context(), u))
		got, err = f.units.GetByID(t.Context(), tenant, u.ID())
		require.NoError(t, err)
		require.Empty(t, got.DIANCode())
	})

	t.Run("el producto trae el código DIAN de su unidad base y de sus unidades alternativas", func(t *testing.T) {
		tenant := f.createTenant(t)
		pair := f.createUnit(t, tenant, "par", "Par")
		require.NoError(t, pair.SetDIANCode("PR"))
		require.NoError(t, f.units.Update(t.Context(), pair))
		box := f.createUnit(t, tenant, "caja", "Caja")
		p := f.createProduct(t, tenant, "TEN-38", "Tenis",
			func(p *domain.NewProductParams) { p.BaseUnit = pair.Ref() })
		require.NoError(t, f.setUnit(t, p, box.Ref(), "6", "500000", ""))

		got, err := f.products.GetByID(t.Context(), tenant, p.ID())
		require.NoError(t, err)
		require.Equal(t, "PR", got.BaseUnit().DIANCode)
		require.Empty(t, got.DefaultVariant().Units()[0].Unit().DIANCode)
	})

	t.Run("guarda y lee el tratamiento del IVA del producto (papelería exenta, droguería excluida)", func(t *testing.T) {
		tenant := f.createTenant(t)
		for _, tt := range []struct {
			sku       string
			treatment vat.Treatment
		}{{"CUA-100", vat.Exempt}, {"ACE-500", vat.Excluded}} {
			p := f.createProduct(t, tenant, tt.sku, "Producto "+tt.sku, withTax(t, "0", tt.treatment))

			got, err := f.products.GetByID(t.Context(), tenant, p.ID())
			require.NoError(t, err)
			require.Equal(t, tt.treatment, got.TaxRate().Treatment())
		}
	})
}
