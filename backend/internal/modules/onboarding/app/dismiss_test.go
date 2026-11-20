package app_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
)

func TestDismissChecklist(t *testing.T) {
	t.Run("ocultar guarda los pasos pendientes de ese momento", func(t *testing.T) {
		for _, template := range []string{"ferreteria", "ropa_calzado"} {
			e := newEnv(t, template)
			e.state.facts = domain.Facts{HasContact: true, HasProduct: true}

			require.NoError(t, e.svc.DismissChecklist(t.Context(), e.actor))

			d, ok, err := e.dismissals.Get(t.Context(), e.actor.TenantID, e.actor.UserID)
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, now, d.DismissedAt())
			require.NotContains(t, d.Pending(), domain.StepProducts)
			require.Contains(t, d.Pending(), domain.StepFirstSale)
			e.span(t, "onboarding.DismissChecklist")
			require.Equal(t, int64(1), e.counted(t, "cuadra.onboarding.dismissals"))
		}
	})
	t.Run("oculta la lista solo para quien la ocultó", func(t *testing.T) {
		e := newEnv(t, "ferreteria")
		require.NoError(t, e.svc.DismissChecklist(t.Context(), e.actor))

		mine, err := e.svc.GetChecklist(t.Context(), e.actor)
		require.NoError(t, err)
		require.False(t, mine.Visible())

		other := e.actor
		other.UserID = uuid.Must(uuid.NewV7())
		theirs, err := e.svc.GetChecklist(t.Context(), other)
		require.NoError(t, err)
		require.True(t, theirs.Visible())
	})
	t.Run("vuelve a aparecer si la resolución vigente se vence", func(t *testing.T) {
		e := newEnv(t, "ropa_calzado")
		e.invoicing.readiness = app.InvoicingReadiness{Enabled: true, Ready: true}
		require.NoError(t, e.svc.DismissChecklist(t.Context(), e.actor))

		e.invoicing.readiness.Ready = false
		c, err := e.svc.GetChecklist(t.Context(), e.actor)
		require.NoError(t, err)
		require.True(t, c.Dismissed)
		require.True(t, c.Visible())
	})
	t.Run("no oculta una lista sin pasos para el rol", func(t *testing.T) {
		e := newEnv(t, "ferreteria")
		e.actor.Role = "warehouse"

		err := e.svc.DismissChecklist(t.Context(), e.actor)
		require.ErrorIs(t, err, domain.ErrNothingToDismiss)
	})
}
