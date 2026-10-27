package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func ruleSpec(minQty, maxQty string) domain.ReorderRuleSpec {
	return domain.ReorderRuleSpec{
		TenantID: newID(), VariantID: newID(), LocationID: newID(),
		Min: dec(minQty), Max: dec(maxQty),
	}
}

func newRule(t *testing.T, minQty, maxQty string) *domain.ReorderRule {
	t.Helper()
	r, err := domain.NewReorderRule(ruleSpec(minQty, maxQty))
	require.NoError(t, err)
	return r
}

func TestNewReorderRule(t *testing.T) {
	valid := []struct{ name, min, max string }{
		{"unidades sueltas de una tienda", "10", "50"},
		{"pares de calzado", "2", "6"},
		{"metros de un rollo", "2.5", "30.75"},
		{"mínimo igual al máximo", "4", "4"},
		{"mínimo en cero", "0", "12"},
	}
	for _, tc := range valid {
		t.Run("crea la regla: "+tc.name, func(t *testing.T) {
			s := ruleSpec(tc.min, tc.max)
			r, err := domain.NewReorderRule(s)
			require.NoError(t, err)
			require.Equal(t, s.TenantID, r.TenantID())
			require.Equal(t, s.VariantID, r.VariantID())
			require.Equal(t, s.LocationID, r.LocationID())
			requireDec(t, tc.min, r.Min())
			requireDec(t, tc.max, r.Max())
		})
	}

	invalid := []struct{ name, min, max string }{
		{"rechaza mínimo negativo", "-1", "10"},
		{"rechaza máximo menor que el mínimo", "8", "5"},
		{"rechaza más de cuatro decimales", "1.00001", "10"},
		{"rechaza un máximo que no cabe", "1", "100000000000000"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewReorderRule(ruleSpec(tc.min, tc.max))
			require.ErrorIs(t, err, domain.ErrInvalidReorderRange)
		})
	}

	t.Run("rechaza ids vacíos", func(t *testing.T) {
		for field, want := range map[string]error{
			"tenant": domain.ErrInvalidTenantID, "variant": domain.ErrInvalidVariantID, "location": domain.ErrInvalidLocationID,
		} {
			s := ruleSpec("1", "2")
			switch field {
			case "tenant":
				s.TenantID = uuid.Nil
			case "variant":
				s.VariantID = uuid.Nil
			case "location":
				s.LocationID = uuid.Nil
			}
			_, err := domain.NewReorderRule(s)
			require.ErrorIs(t, err, want, field)
		}
	})
}

func TestReorderRuleAlert(t *testing.T) {
	cases := []struct {
		name, min, max, onHand string
		alert                  bool
		suggested              string
	}{
		{"aparece cuando el stock es igual al mínimo", "10", "50", "10", true, "40"},
		{"aparece cuando el stock es menor que el mínimo", "2", "6", "1", true, "5"},
		{"no aparece cuando el stock es mayor que el mínimo", "10", "50", "10.5", false, ""},
		{"sin stock sugiere el máximo", "2", "6", "0", true, "6"},
		{"con stock negativo sugiere el máximo más lo que falta", "5", "20", "-3", true, "23"},
		{"funciona con cantidades fraccionarias", "2.5", "30.75", "1.25", true, "29.5"},
		{"mínimo en cero aparece solo sin stock", "0", "12", "0", true, "12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRule(t, tc.min, tc.max)
			require.Equal(t, tc.alert, r.NeedsReorder(dec(tc.onHand)))
			alert, ok := r.Alert(dec(tc.onHand))
			require.Equal(t, tc.alert, ok)
			if !tc.alert {
				return
			}
			require.Same(t, r, alert.Rule)
			requireDec(t, tc.onHand, alert.OnHand)
			requireDec(t, tc.suggested, alert.Suggested)
		})
	}
}

func TestReorderRuleSuggested(t *testing.T) {
	t.Run("la sugerencia nunca es negativa", func(t *testing.T) {
		for _, onHand := range []string{"6", "7", "1000"} {
			r := newRule(t, "2", "6")
			require.True(t, r.Suggested(dec(onHand)).IsZero(), onHand)
		}
	})

	t.Run("la sugerencia es el máximo menos el stock actual", func(t *testing.T) {
		requireDec(t, "35", newRule(t, "10", "50").Suggested(dec("15")))
	})
}
