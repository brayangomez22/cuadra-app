package app_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestAuditOfTenantSettings(t *testing.T) {
	text := func(s string) *string { return &s }

	t.Run("cambiar el IVA por defecto deja un evento con el anterior y el nuevo", func(t *testing.T) {
		e := newEnv(t)
		tenant, owner := e.seed(t, "ana@drogueria.co", validPassword, domain.RoleOwner)
		rate := decimal.RequireFromString("0.05")

		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), owner.ID(), app.TenantUpdate{DefaultTaxRate: &rate})
		require.NoError(t, err)

		events := e.audit.of("tenant.settings_changed")
		require.Len(t, events, 1)
		require.Equal(t, tenant.ID(), events[0].EntityID)
		require.Equal(t, map[string]string{"setting": "default_tax_rate", "old": "0.19", "new": "0.05"}, events[0].Details)
	})

	t.Run("deja un evento por cada dato que cambia, y ninguno por los iguales", func(t *testing.T) {
		e := newEnv(t)
		tenant, owner := e.seed(t, "ana@calzado.co", validPassword, domain.RoleOwner)
		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), owner.ID(), app.TenantUpdate{City: text("Medellín"), Phone: text("604 444 5566")})
		require.NoError(t, err)
		e.audit.entries = nil

		_, err = e.svc.UpdateTenant(t.Context(), tenant.ID(), owner.ID(), app.TenantUpdate{
			City: text("Medellín"), Phone: text(""), Address: text("Cra 50 # 10-20"),
		})
		require.NoError(t, err)

		events := e.audit.of("tenant.settings_changed")
		require.Len(t, events, 2)
		require.Equal(t, map[string]string{"setting": "address", "old": "", "new": "Cra 50 # 10-20"}, events[0].Details)
		require.Equal(t, map[string]string{"setting": "phone", "old": "604 444 5566", "new": ""}, events[1].Details)
	})

	t.Run("si el registro falla, la configuración no cambia", func(t *testing.T) {
		e := newEnv(t)
		tenant, owner := e.seed(t, "ana@papeleria.co", validPassword, domain.RoleOwner)
		e.audit.fail = errors.New("audit down")
		rate := decimal.RequireFromString("0.05")

		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), owner.ID(), app.TenantUpdate{DefaultTaxRate: &rate})
		require.ErrorContains(t, err, "audit down")
	})
}
