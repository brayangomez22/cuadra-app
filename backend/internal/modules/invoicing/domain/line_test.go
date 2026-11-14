package domain_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestLineTaxes(t *testing.T) {
	cases := []struct {
		name            string
		treatment       vat.Treatment
		rate, base, tax string
		want            []domain.LineTax
	}{
		{"un producto excluido no lleva IVA en el documento (medicamento de droguería)",
			vat.Excluded, "0", "12000.00", "0.00", nil},
		{"un producto exento lleva IVA con tasa 0 (cuaderno de papelería)",
			vat.Exempt, "0", "4500.00", "0.00",
			[]domain.LineTax{{Code: domain.TaxCodeIVA, Rate: dec("0"), Base: dec("4500.00"), Amount: dec("0.00")}}},
		{"un producto gravado lleva IVA a su tasa (cable de ferretería)",
			vat.Taxed, "0.19", "10000.00", "1900.00",
			[]domain.LineTax{{Code: domain.TaxCodeIVA, Rate: dec("0.19"), Base: dec("10000.00"), Amount: dec("1900.00")}}},
		{"un producto gravado al 5 % lleva su tasa (almacén de ropa)",
			vat.Taxed, "0.05", "80000.00", "4000.00",
			[]domain.LineTax{{Code: domain.TaxCodeIVA, Rate: dec("0.05"), Base: dec("80000.00"), Amount: dec("4000.00")}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := domain.LineTaxes(c.treatment, dec(c.rate), dec(c.base), dec(c.tax))
			require.Equal(t, c.want, got)
		})
	}
}

func TestUnitCode(t *testing.T) {
	cases := []struct{ name, dian, want string }{
		{"una unidad sin código DIAN usa el de unidad (94): la caja de la papelería", "", "94"},
		{"una unidad con código DIAN lo conserva: el metro de la ferretería", "MTR", "MTR"},
		{"una unidad con código DIAN lo conserva: el par del almacén de calzado", "PR", "PR"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, domain.UnitCode(c.dian))
		})
	}
}

func TestLineCode(t *testing.T) {
	t.Run("la línea va con el SKU de la variante", func(t *testing.T) {
		require.Equal(t, "CAB-12", domain.LineCode("CAB-12", "0190a0b1-0000-7000-8000-000000000001"))
	})
	t.Run("una línea guardada sin SKU va con el id de la variante", func(t *testing.T) {
		require.Equal(t, "0190a0b1-0000-7000-8000-000000000001", domain.LineCode("", "0190a0b1-0000-7000-8000-000000000001"))
	})
}
