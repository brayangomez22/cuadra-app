package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// demoTenant sets up the demo tenant of a demoCases entry.
func (e *env) demoTenant(t *testing.T, name, template, mailbox string) uuid.UUID {
	t.Helper()
	id, err := e.svc.SetUpDemoTenant(t.Context(), demoInput(name, template, mailbox))
	require.NoError(t, err)
	return id
}

func demoRequest(template string, role domain.Role) app.DemoLoginInput {
	return app.DemoLoginInput{BusinessTemplate: template, Role: role.String(), ClientIP: "203.0.113.7"}
}

func TestStartDemo(t *testing.T) {
	roles := []domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleCashier, domain.RoleWarehouse}
	for _, c := range demoCases {
		for _, role := range roles {
			t.Run("abre la sesión del rol elegido en la demo: "+c.template+" "+role.String(), func(t *testing.T) {
				e := newEnv(t)
				tenantID := e.demoTenant(t, c.name, c.template, c.mailbox)

				session, err := e.svc.StartDemo(t.Context(), demoRequest(c.template, role))
				require.NoError(t, err)
				require.Equal(t, tenantID, session.Profile.Tenant.ID())
				require.Equal(t, role, session.Profile.User.Role())
				require.True(t, session.Profile.Tenant.IsDemo())
				require.NotEmpty(t, session.AccessToken)
				requireUsableRefresh(t, e, session.RefreshToken, session.Profile.User.ID())

				span := e.span(t, "identity.StartDemo")
				require.Contains(t, span.Attributes(), attribute.String("tenant.id", tenantID.String()))
				require.Contains(t, span.Attributes(), attribute.String("user.id", session.Profile.User.ID().String()))
				require.Equal(t, int64(1), e.counter(t, "cuadra.identity.demo_logins", "role", role.String()))
			})
		}
	}

	t.Run("no hace hash ni pide contraseña: entra sin verificar ninguna", func(t *testing.T) {
		e := newEnv(t)
		e.demoTenant(t, "Ferretería Demo", "ferreteria", "ferreteria.demo.test")
		before := e.hasher.Verifies()
		_, err := e.svc.StartDemo(t.Context(), demoRequest("ferreteria", domain.RoleCashier))
		require.NoError(t, err)
		require.Equal(t, before, e.hasher.Verifies())
	})
}

func TestStartDemoRejects(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T, e *env) app.DemoLoginInput
		opts    []func(*app.Deps)
	}{
		{
			name: "rechaza con la demo apagada",
			opts: []func(*app.Deps){func(d *app.Deps) { d.DemoEnabled = false }},
			prepare: func(t *testing.T, e *env) app.DemoLoginInput {
				t.Helper()
				e.demoTenant(t, "Ferretería Demo", "ferreteria", "ferreteria.demo.test")
				return demoRequest("ferreteria", domain.RoleCashier)
			},
		},
		{
			name: "rechaza un tenant que no es demo aunque el repositorio lo devuelva",
			prepare: func(t *testing.T, e *env) app.DemoLoginInput {
				t.Helper()
				tenant, _ := e.seed(t, "ana@tienda.test", validPassword, domain.RoleOwner)
				e.demo.extra = []uuid.UUID{tenant.ID()}
				return demoRequest("ferreteria", domain.RoleOwner)
			},
		},
		{
			name: "rechaza un tipo de negocio sin demo",
			prepare: func(t *testing.T, e *env) app.DemoLoginInput {
				t.Helper()
				e.demoTenant(t, "Ferretería Demo", "ferreteria", "ferreteria.demo.test")
				return demoRequest("drogueria", domain.RoleCashier)
			},
		},
		{
			name: "rechaza un rol sin usuario activo en la demo",
			prepare: func(t *testing.T, e *env) app.DemoLoginInput {
				t.Helper()
				id := e.demoTenant(t, "Almacén de Ropa Demo", "ropa_calzado", "ropa.demo.test")
				for _, u := range e.users.byID {
					if u.TenantID() == id && u.Role() == domain.RoleWarehouse {
						u.Deactivate()
					}
				}
				return demoRequest("ropa_calzado", domain.RoleWarehouse)
			},
		},
		{
			name: "rechaza un rol que no existe",
			prepare: func(t *testing.T, e *env) app.DemoLoginInput {
				t.Helper()
				e.demoTenant(t, "Ferretería Demo", "ferreteria", "ferreteria.demo.test")
				return app.DemoLoginInput{BusinessTemplate: "ferreteria", Role: "superuser"}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t, c.opts...)
			in := c.prepare(t, e)
			tokens := len(e.tokens.byID)

			_, err := e.svc.StartDemo(t.Context(), in)
			require.ErrorIs(t, err, domain.ErrDemoUnavailable)
			require.Len(t, e.tokens.byID, tokens, "no session is started")
			require.Equal(t, int64(1), e.counter(t, "cuadra.identity.demo_logins", "result", "failure"))
		})
	}

	t.Run("limita los intentos por IP", func(t *testing.T) {
		e := newEnv(t)
		e.demoTenant(t, "Ferretería Demo", "ferreteria", "ferreteria.demo.test")
		e.demoLimiter.denied["203.0.113.7"] = true

		_, err := e.svc.StartDemo(t.Context(), demoRequest("ferreteria", domain.RoleCashier))
		var limited *app.RateLimitError
		require.ErrorAs(t, err, &limited)
		require.Equal(t, time.Minute, limited.RetryAfter)
		require.Empty(t, e.tokens.byID)
	})
}

func TestDemoOptions(t *testing.T) {
	t.Run("lista solo los tipos de negocio con demo, en el orden de las plantillas", func(t *testing.T) {
		e := newEnv(t)
		e.demoTenant(t, "Almacén de Ropa Demo", "ropa_calzado", "ropa.demo.test")
		e.demoTenant(t, "Ferretería Demo", "ferreteria", "ferreteria.demo.test")
		e.seed(t, "ana@drogueria.test", validPassword, domain.RoleOwner)

		options, err := e.svc.DemoOptions(t.Context())
		require.NoError(t, err)
		require.Equal(t, []app.BusinessTemplate{testTemplates[1], testTemplates[2]}, options)
		require.NotNil(t, e.span(t, "identity.ListDemos"))
	})

	t.Run("con la demo apagada responde que no está disponible", func(t *testing.T) {
		e := newEnv(t, func(d *app.Deps) { d.DemoEnabled = false })
		e.demoTenant(t, "Ferretería Demo", "ferreteria", "ferreteria.demo.test")

		_, err := e.svc.DemoOptions(t.Context())
		require.ErrorIs(t, err, domain.ErrDemoUnavailable)
	})
}
