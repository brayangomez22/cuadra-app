package vat_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/vat"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name      string
		treatment vat.Treatment
		zeroRate  bool
		want      vat.Treatment
		err       error
	}{
		{"deduce gravado con tasa mayor que cero (taladro de ferretería al 19 %)", "", false, vat.Taxed, nil},
		{"deduce excluido con tasa cero (medicamento de droguería)", "", true, vat.Excluded, nil},
		{"acepta exento con tasa cero (cuaderno de papelería)", vat.Exempt, true, vat.Exempt, nil},
		{"acepta excluido con tasa cero", vat.Excluded, true, vat.Excluded, nil},
		{"acepta gravado con tasa positiva (camisa de almacén de ropa)", vat.Taxed, false, vat.Taxed, nil},
		{"rechaza gravado con tasa cero", vat.Taxed, true, "", vat.ErrInvalidTreatment},
		{"rechaza exento con tasa positiva", vat.Exempt, false, "", vat.ErrInvalidTreatment},
		{"rechaza excluido con tasa positiva", vat.Excluded, false, "", vat.ErrInvalidTreatment},
		{"rechaza un tratamiento desconocido", "zero_rated", true, "", vat.ErrInvalidTreatment},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := vat.Classify(c.treatment, c.zeroRate)
			require.ErrorIs(t, err, c.err)
			require.Equal(t, c.want, got)
		})
	}
}

func TestTreatmentCharges(t *testing.T) {
	t.Run("solo el excluido no lleva IVA en el documento", func(t *testing.T) {
		require.True(t, vat.Taxed.Charged())
		require.True(t, vat.Exempt.Charged())
		require.False(t, vat.Excluded.Charged())
	})
}
