package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
)

func ids(c domain.Checklist) []domain.StepID {
	out := make([]domain.StepID, 0, len(c.Steps))
	for _, s := range c.Steps {
		out = append(out, s.ID)
	}
	return out
}

func TestGetChecklist(t *testing.T) {
	t.Run("arma la lista con los pasos de la plantilla de la empresa", func(t *testing.T) {
		hardware, clothing := newEnv(t, "ferreteria"), newEnv(t, "ropa_calzado")

		h, err := hardware.svc.GetChecklist(t.Context(), hardware.actor)
		require.NoError(t, err)
		c, err := clothing.svc.GetChecklist(t.Context(), clothing.actor)
		require.NoError(t, err)

		require.Contains(t, ids(h), domain.StepCustomerCredit)
		require.Contains(t, ids(c), domain.StepVariantProducts)
		require.NotEqual(t, ids(h), ids(c))
		require.True(t, h.Visible())
		hardware.span(t, "onboarding.GetChecklist")
	})
	t.Run("lee la facturación del módulo de facturación", func(t *testing.T) {
		e := newEnv(t, "ferreteria")
		e.invoicing.readiness = app.InvoicingReadiness{Enabled: true}

		c, err := e.svc.GetChecklist(t.Context(), e.actor)
		require.NoError(t, err)
		require.Contains(t, ids(c), domain.StepInvoicing)

		e.invoicing.readiness.Ready = true
		c, err = e.svc.GetChecklist(t.Context(), e.actor)
		require.NoError(t, err)
		require.Equal(t, domain.StepInvoicing, c.Steps[1].ID)
		require.True(t, c.Steps[1].Done)
	})
	t.Run("solo pregunta a facturación si el rol puede resolver ese paso", func(t *testing.T) {
		e := newEnv(t, "ropa_calzado")
		e.actor.Role = "cashier"
		e.perms = []string{"sales:create"}
		e.invoicing.readiness = app.InvoicingReadiness{Enabled: true}

		c, err := e.svc.GetChecklist(t.Context(), e.actor)
		require.NoError(t, err)
		require.Equal(t, []domain.StepID{domain.StepFirstSale}, ids(c))
	})
	t.Run("una plantilla que ya no existe da una lista vacía", func(t *testing.T) {
		e := newEnv(t, "retirada")

		c, err := e.svc.GetChecklist(t.Context(), e.actor)
		require.NoError(t, err)
		require.Empty(t, c.Steps)
		require.False(t, c.Visible())
	})
	t.Run("con todo hecho la lista no se muestra", func(t *testing.T) {
		e := newEnv(t, "ferreteria")
		e.state.facts = domain.Facts{
			HasContact: true, HasProduct: true, HasAlternateUnit: true, HasStock: true,
			HasCreditCustomer: true, HasTeammate: true, HasSale: true,
		}

		c, err := e.svc.GetChecklist(t.Context(), e.actor)
		require.NoError(t, err)
		require.False(t, c.Visible())
	})
}
