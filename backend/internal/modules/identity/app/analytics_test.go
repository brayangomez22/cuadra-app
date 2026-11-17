package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestAnalyticsConsent(t *testing.T) {
	yes, no := true, false

	t.Run("una empresa nueva no envía analítica hasta que el dueño acepta", func(t *testing.T) {
		e := newEnv(t)
		tenant, owner := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)
		require.False(t, tenant.AnalyticsConsent())

		updated, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), owner.ID(), app.TenantUpdate{AnalyticsConsent: &yes})
		require.NoError(t, err)
		require.True(t, updated.AnalyticsConsent())

		events := e.audit.of("tenant.settings_changed")
		require.Len(t, events, 1)
		require.Equal(t, map[string]string{"setting": "analytics_consent", "old": "false", "new": "true"}, events[0].Details)
	})

	t.Run("retirar el consentimiento lo apaga", func(t *testing.T) {
		e := newEnv(t)
		tenant, owner := e.seed(t, "ana@modaurbana.co", validPassword, domain.RoleOwner)
		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), owner.ID(), app.TenantUpdate{AnalyticsConsent: &yes})
		require.NoError(t, err)

		updated, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), owner.ID(), app.TenantUpdate{AnalyticsConsent: &no})
		require.NoError(t, err)
		require.False(t, updated.AnalyticsConsent())
	})
}

func TestAnalyticsID(t *testing.T) {
	yes := true
	cases := []struct {
		name    string
		consent bool
		flag    bool
		enabled bool
		want    bool
	}{
		{"con consentimiento, la flag prendida y la analítica configurada trae el seudónimo", true, true, true, true},
		{"sin el consentimiento de la empresa no trae seudónimo", false, true, true, false},
		{"con la flag apagada no trae seudónimo", true, false, true, false},
		{"sin la analítica configurada no trae seudónimo", true, true, false, false},
	}
	// A hardware store and a shoe store: the rule does not depend on the business.
	for business, email := range map[string]string{"ferretería": "ana@eltornillo.co", "calzado": "ana@calzadoandino.co"} {
		for _, tc := range cases {
			t.Run(business+": "+tc.name, func(t *testing.T) {
				e := newEnv(t)
				e.pseudonyms.enabled = tc.enabled
				tenant, owner := e.seed(t, email, validPassword, domain.RoleOwner)
				if tc.consent {
					_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), owner.ID(), app.TenantUpdate{AnalyticsConsent: &yes})
					require.NoError(t, err)
				}
				if tc.flag {
					e.flags.turnOn(tenant.ID(), "analytics")
				}

				profile, err := e.svc.Me(t.Context(), tenant.ID(), owner.ID())
				require.NoError(t, err)

				if tc.want {
					require.Equal(t, "seud-"+owner.ID().String(), profile.AnalyticsID)
					require.NotContains(t, profile.AnalyticsID, owner.Email().String(), "nunca el correo")
				} else {
					require.Empty(t, profile.AnalyticsID)
				}
			})
		}
	}
}
