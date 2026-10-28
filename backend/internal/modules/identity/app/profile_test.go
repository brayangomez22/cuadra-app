package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestMe(t *testing.T) {
	t.Run("devuelve el usuario y su tenant", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)

		profile, err := e.svc.Me(t.Context(), tenant.ID(), user.ID())

		require.NoError(t, err)
		require.Equal(t, user.ID(), profile.User.ID())
		require.Equal(t, tenant.ID(), profile.Tenant.ID())
		e.span(t, "identity.GetMe")
	})

	t.Run("un usuario de otro tenant no se encuentra", func(t *testing.T) {
		e := newEnv(t)
		_, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		other, _ := e.seed(t, "otra@eltornillo.co", validPassword, domain.RoleCashier)

		_, err := e.svc.Me(t.Context(), other.ID(), user.ID())

		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("usuario desactivado devuelve ErrUserNotFound", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		user.Deactivate()
		require.NoError(t, e.users.Update(t.Context(), user))

		_, err := e.svc.Me(t.Context(), tenant.ID(), user.ID())

		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("lista los módulos activos del tenant", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "bodega@eltornillo.co", validPassword, domain.RoleWarehouse)
		other, _ := e.seed(t, "bodega@modaurbana.co", validPassword, domain.RoleWarehouse)
		e.modules.activate(tenant.ID(), "variants", "lots_expiry")
		e.modules.activate(other.ID(), "work_orders")

		profile, err := e.svc.Me(t.Context(), tenant.ID(), user.ID())

		require.NoError(t, err)
		require.Equal(t, []string{"variants", "lots_expiry"}, profile.Modules)
	})

	t.Run("los permisos de un módulo inactivo no aparecen en /me", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "bodega@eltornillo.co", validPassword, domain.RoleWarehouse)

		profile, err := e.svc.Me(t.Context(), tenant.ID(), user.ID())
		require.NoError(t, err)
		require.Equal(t, []domain.Permission{
			domain.PermCatalogRead, domain.PermCustomersRead, domain.PermInventoryRead, domain.PermInventoryAdjust, domain.PermInventoryCount,
		}, profile.Permissions)

		e.modules.activate(tenant.ID(), "lots_expiry")
		profile, err = e.svc.Me(t.Context(), tenant.ID(), user.ID())
		require.NoError(t, err)
		require.Equal(t, []domain.Permission{
			domain.PermCatalogRead, domain.PermCustomersRead, domain.PermInventoryRead, domain.PermInventoryAdjust, domain.PermInventoryCount, "lots:manage",
		}, profile.Permissions)
	})
}

func TestNewPermissionMatrix(t *testing.T) {
	t.Run("registra los permisos de los módulos junto a los del core", func(t *testing.T) {
		m, err := app.NewPermissionMatrix(testRegistry)

		require.NoError(t, err)
		require.True(t, m.Can(domain.RoleWarehouse, "lots:manage"))
		require.False(t, m.Can(domain.RoleCashier, "lots:manage"))
		require.True(t, m.Can(domain.RoleCashier, domain.PermSalesCreate))
	})

	t.Run("un módulo que declara un rol desconocido impide arrancar", func(t *testing.T) {
		_, err := app.NewPermissionMatrix(fakeRegistry{{Module: "lots_expiry", Permission: "lots:manage", Roles: []string{"root"}}})
		require.ErrorIs(t, err, domain.ErrInvalidRole)
	})

	t.Run("un módulo que repite un permiso del core impide arrancar", func(t *testing.T) {
		_, err := app.NewPermissionMatrix(fakeRegistry{{Module: "variants", Permission: "catalog:write", Roles: []string{"admin"}}})
		require.ErrorIs(t, err, domain.ErrDuplicatePermission)
	})

	t.Run("un módulo que declara un permiso mal escrito impide arrancar", func(t *testing.T) {
		_, err := app.NewPermissionMatrix(fakeRegistry{{Module: "lots_expiry", Permission: "Lots manage", Roles: []string{"admin"}}})
		require.ErrorIs(t, err, domain.ErrInvalidPermission)
	})
}
