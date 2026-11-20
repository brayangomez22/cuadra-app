package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
)

func TestDismissal(t *testing.T) {
	tenantID, userID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	at := time.Date(2026, 11, 19, 15, 0, 0, 0, time.UTC)
	modules := []string{"unit_conversions", "variants"}
	ready := domain.Facts{HasContact: true, InvoicingEnabled: true, InvoicingReady: true}
	build := func(steps []string, facts domain.Facts, d *domain.Dismissal) domain.Checklist {
		return domain.BuildChecklist(domain.ChecklistInput{Steps: steps, Facts: facts, Can: owner, Modules: modules, Dismissal: d})
	}
	dismiss := func(t *testing.T, steps []string, facts domain.Facts) *domain.Dismissal {
		t.Helper()
		d, err := domain.NewDismissal(tenantID, userID, build(steps, facts, nil), at)
		require.NoError(t, err)
		return &d
	}

	t.Run("ocultar guarda los pasos pendientes de ese momento", func(t *testing.T) {
		d := dismiss(t, clothingSteps, ready)
		require.Equal(t, []domain.StepID{
			domain.StepProducts, domain.StepVariantProducts, domain.StepInitialStock, domain.StepInviteTeam, domain.StepFirstSale,
		}, d.Pending())
		require.Equal(t, at, d.DismissedAt())
		require.Equal(t, userID, d.UserID())
	})
	t.Run("ocultada, no vuelve por un paso no crítico pendiente", func(t *testing.T) {
		for _, steps := range [][]string{hardwareSteps, clothingSteps} {
			c := build(steps, ready, dismiss(t, steps, ready))
			require.True(t, c.Dismissed)
			require.False(t, c.Visible())
		}
	})
	t.Run("ocultada con la facturación pendiente, no vuelve por ella", func(t *testing.T) {
		pending := domain.Facts{InvoicingEnabled: true}
		c := build(hardwareSteps, pending, dismiss(t, hardwareSteps, pending))
		require.False(t, c.Visible())
	})
	t.Run("ocultada, vuelve si falta un paso crítico que estaba completo (resolución vencida)", func(t *testing.T) {
		for _, steps := range [][]string{hardwareSteps, clothingSteps} {
			d := dismiss(t, steps, ready)
			expired := ready
			expired.InvoicingReady = false
			c := build(steps, expired, d)
			require.True(t, c.Visible())
		}
	})
	t.Run("ocultada antes de activar la emisión, vuelve al activarla sin resolución", func(t *testing.T) {
		d := dismiss(t, hardwareSteps, domain.Facts{})
		require.True(t, build(hardwareSteps, domain.Facts{InvoicingEnabled: true}, d).Visible())
	})
	t.Run("no se oculta una lista sin pasos", func(t *testing.T) {
		empty := domain.BuildChecklist(domain.ChecklistInput{Steps: clothingSteps, Can: only()})
		_, err := domain.NewDismissal(tenantID, userID, empty, at)
		require.ErrorIs(t, err, domain.ErrNothingToDismiss)
	})
	t.Run("rehidrata un ocultamiento guardado", func(t *testing.T) {
		d := domain.RestoreDismissal(tenantID, userID, at, []domain.StepID{domain.StepFirstSale})
		require.Equal(t, tenantID, d.TenantID())
		require.Equal(t, []domain.StepID{domain.StepFirstSale}, d.Pending())
	})
}
