package domain_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSaleLineCodes(t *testing.T) {
	tests := []struct {
		name                     string
		sku, baseUnit, unitDIAN  string
		unit                     domain.LineUnit
		wantSKU, wantBase, wantD string
	}{
		{"ferretería: la línea congela el SKU, la unidad base y su código DIAN",
			" cab-12 ", " m ", "MTR", domain.BaseUnit, "cab-12", "m", "MTR"},
		{"almacén de calzado: la línea en unidad base sin código DIAN",
			"TEN-38-NEG", "par", "", domain.BaseUnit, "TEN-38-NEG", "par", ""},
		{"ferretería: la línea en unidad alternativa congela el código DIAN de esa unidad",
			"TOR-1", "und", "94", domain.LineUnit{Code: "caja", Factor: dec("100")}, "TOR-1", "und", "94"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newSale(t)
			in := line("1", "1000", "0.19", "")
			in.SKU, in.BaseUnitCode, in.UnitDIANCode, in.Unit = tt.sku, tt.baseUnit, tt.unitDIAN, tt.unit
			addLine(t, s, in)

			l := firstLine(t, s)
			require.Equal(t, tt.wantSKU, l.SKU())
			require.Equal(t, tt.wantBase, l.BaseUnitCode())
			require.Equal(t, tt.wantD, l.UnitDIANCode())
		})
	}

	t.Run("rechaza códigos demasiado largos", func(t *testing.T) {
		for _, mutate := range []func(*domain.LineInput){
			func(in *domain.LineInput) { in.SKU = strings.Repeat("A", 65) },
			func(in *domain.LineInput) { in.BaseUnitCode = strings.Repeat("u", 17) },
			func(in *domain.LineInput) { in.UnitDIANCode = "KGMS" },
		} {
			in := line("1", "1000", "0.19", "")
			mutate(&in)
			require.ErrorIs(t, newSale(t).AddLine(in, domain.NoDiscountLimit), domain.ErrInvalidLineCode)
		}
	})
}

func TestSaleLineTaxTreatment(t *testing.T) {
	tests := []struct {
		name      string
		rate      string
		treatment vat.Treatment
		want      vat.Treatment
	}{
		{"la línea congela el tratamiento exento (papelería)", "0", vat.Exempt, vat.Exempt},
		{"la línea congela el tratamiento excluido (droguería)", "0", vat.Excluded, vat.Excluded},
		{"la línea gravada (almacén de ropa)", "0.19", vat.Taxed, vat.Taxed},
		{"una línea guardada sin tratamiento lo deduce de su tasa", "0", "", vat.Excluded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newSale(t)
			in := line("1", "1000", tt.rate, "")
			in.TaxTreatment = tt.treatment
			addLine(t, s, in)
			require.Equal(t, tt.want, firstLine(t, s).TaxTreatment())
		})
	}

	t.Run("rechaza un tratamiento que no corresponde a la tasa", func(t *testing.T) {
		in := line("1", "1000", "0.19", "")
		in.TaxTreatment = vat.Exempt
		require.ErrorIs(t, newSale(t).AddLine(in, domain.NoDiscountLimit), domain.ErrInvalidTaxTreatment)
	})

	t.Run("la venta guardada conserva códigos y tratamiento", func(t *testing.T) {
		s := newSale(t)
		in := line("2", "1000", "0", "")
		in.SKU, in.BaseUnitCode, in.UnitDIANCode, in.TaxTreatment = "CUA-100", "und", "94", vat.Exempt
		addLine(t, s, in)
		snap := s.Snapshot().Lines[0]
		require.Equal(t, "CUA-100", snap.SKU)
		require.Equal(t, "und", snap.BaseUnitCode)
		require.Equal(t, "94", snap.UnitDIANCode)
		require.Equal(t, vat.Exempt, snap.TaxTreatment)
	})
}
