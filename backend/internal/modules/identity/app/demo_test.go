package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// demoInput is a demo tenant of template with one user per role.
func demoInput(name, template, mailbox string) app.DemoTenantInput {
	return app.DemoTenantInput{
		TenantName:       name,
		NIT:              validNIT,
		BusinessTemplate: template,
		Password:         validPassword,
		Owner:            app.DemoUser{Name: "Dueño Demo", Email: "dueno@" + mailbox},
		Staff: []app.DemoUser{
			{Name: "Administrador Demo", Email: "admin@" + mailbox, Role: domain.RoleAdmin},
			{Name: "Cajero Demo", Email: "cajero@" + mailbox, Role: domain.RoleCashier},
			{Name: "Bodeguero Demo", Email: "bodega@" + mailbox, Role: domain.RoleWarehouse},
		},
	}
}

var demoCases = []struct{ name, template, mailbox string }{
	{"Ferretería Demo", "ferreteria", "ferreteria.demo.test"},
	{"Almacén de Ropa Demo", "ropa_calzado", "ropa.demo.test"},
}

func TestSetUpDemoTenant(t *testing.T) {
	for _, c := range demoCases {
		t.Run("crea el tenant demo con un usuario por rol: "+c.template, func(t *testing.T) {
			e := newEnv(t)
			id, err := e.svc.SetUpDemoTenant(t.Context(), demoInput(c.name, c.template, c.mailbox))
			require.NoError(t, err)

			tenant, err := e.tenants.GetByID(t.Context(), id)
			require.NoError(t, err)
			require.True(t, tenant.IsDemo())
			require.Equal(t, c.template, tenant.BusinessTemplate())
			require.Equal(t, []string{c.template}, e.setup.templates)
			require.Equal(t, []bool{true}, e.setup.inTx)

			roles := map[string]domain.Role{}
			for _, u := range e.users.byID {
				require.Equal(t, id, u.TenantID())
				roles[u.Email().String()] = u.Role()
			}
			require.Equal(t, map[string]domain.Role{
				"dueno@" + c.mailbox: domain.RoleOwner, "admin@" + c.mailbox: domain.RoleAdmin,
				"cajero@" + c.mailbox: domain.RoleCashier, "bodega@" + c.mailbox: domain.RoleWarehouse,
			}, roles)
			require.NotNil(t, e.span(t, "identity.SetUpDemoTenant"))
		})
	}

	t.Run("rechaza una plantilla que no existe sin guardar nada", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SetUpDemoTenant(t.Context(), demoInput("Demo", "no_existe", "x.demo.test"))
		require.ErrorIs(t, err, domain.ErrUnknownBusinessTemplate)
		require.Empty(t, e.tenants.byID)
	})

	t.Run("rechaza un empleado con rol de dueño", func(t *testing.T) {
		e := newEnv(t)
		in := demoInput("Ferretería Demo", "ferreteria", "ferreteria.demo.test")
		in.Staff[0].Role = domain.RoleOwner
		_, err := e.svc.SetUpDemoTenant(t.Context(), in)
		require.ErrorIs(t, err, domain.ErrInvalidRole)
		require.Empty(t, e.tenants.byID)
	})
}

func TestDemoTenants(t *testing.T) {
	t.Run("lista solo los tenants demo de la plantilla", func(t *testing.T) {
		e := newEnv(t)
		hardware, err := e.svc.SetUpDemoTenant(t.Context(), demoInput("Ferretería Demo", "ferreteria", "ferreteria.demo.test"))
		require.NoError(t, err)
		_, err = e.svc.SetUpDemoTenant(t.Context(), demoInput("Almacén de Ropa Demo", "ropa_calzado", "ropa.demo.test"))
		require.NoError(t, err)
		e.seed(t, "ana@ferreteria.test", validPassword, domain.RoleOwner) // a real tenant of the same template

		ids, err := e.svc.DemoTenants(t.Context(), "ferreteria")
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{hardware}, ids)
	})
}

func TestPurgeDemoTenant(t *testing.T) {
	for _, c := range demoCases {
		t.Run("borra el tenant demo con sus usuarios: "+c.template, func(t *testing.T) {
			e := newEnv(t)
			id, err := e.svc.SetUpDemoTenant(t.Context(), demoInput(c.name, c.template, c.mailbox))
			require.NoError(t, err)

			require.NoError(t, e.svc.PurgeDemoTenant(t.Context(), id))
			require.Equal(t, []uuid.UUID{id}, e.demo.purged)
			require.Empty(t, e.users.byID)
			require.NotNil(t, e.span(t, "identity.PurgeDemoTenant"))
		})
	}

	t.Run("rechaza borrar un tenant que no es demo", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "ana@ferreteria.test", validPassword, domain.RoleOwner)

		err := e.svc.PurgeDemoTenant(t.Context(), tenant.ID())
		require.ErrorIs(t, err, domain.ErrNotDemoTenant)
		require.Empty(t, e.demo.purged)
		_, err = e.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
	})

	t.Run("rechaza borrar un tenant que no existe", func(t *testing.T) {
		e := newEnv(t)
		err := e.svc.PurgeDemoTenant(t.Context(), uuid.Must(uuid.NewV7()))
		require.ErrorIs(t, err, domain.ErrTenantNotFound)
	})
}
