package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestParseRole(t *testing.T) {
	t.Run("acepta los cuatro roles válidos", func(t *testing.T) {
		for _, in := range []string{"owner", "admin", "cashier", "warehouse"} {
			got, err := domain.ParseRole(in)
			require.NoError(t, err, in)
			require.Equal(t, in, got.String())
		}
	})

	t.Run("rechaza rol desconocido", func(t *testing.T) {
		for _, in := range []string{"", "root", "Owner", " admin"} {
			_, err := domain.ParseRole(in)
			require.ErrorIs(t, err, domain.ErrInvalidRole, in)
		}
	})
}

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		name string
		role domain.Role
		perm domain.Permission
		want bool
	}{
		{name: "el cajero no tiene catalog:write", role: domain.RoleCashier, perm: domain.PermCatalogWrite, want: false},
		{name: "el cajero puede vender", role: domain.RoleCashier, perm: domain.PermSalesCreate, want: true},
		{name: "el bodeguero puede ajustar inventario", role: domain.RoleWarehouse, perm: domain.PermInventoryAdjust, want: true},
		{name: "el bodeguero no puede vender", role: domain.RoleWarehouse, perm: domain.PermSalesCreate, want: false},
		{name: "el admin gestiona usuarios", role: domain.RoleAdmin, perm: domain.PermUsersManage, want: true},
		{name: "el admin no puede gestionar la empresa", role: domain.RoleAdmin, perm: domain.PermTenantManage, want: false},
		{name: "un rol desconocido no tiene permisos", role: domain.Role("root"), perm: domain.PermCatalogRead, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.role.Can(tt.perm))
		})
	}

	t.Run("el owner tiene todos los permisos", func(t *testing.T) {
		all := domain.AllPermissions()
		require.NotEmpty(t, all)
		for _, p := range all {
			require.True(t, domain.RoleOwner.Can(p), p)
		}
		require.ElementsMatch(t, all, domain.RoleOwner.Permissions())
	})

	t.Run("todos los roles leen catálogo e inventario", func(t *testing.T) {
		for _, r := range []domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleCashier, domain.RoleWarehouse} {
			require.True(t, r.Can(domain.PermCatalogRead), r)
			require.True(t, r.Can(domain.PermInventoryRead), r)
		}
	})

	t.Run("modificar la lista de permisos devuelta no altera la matriz", func(t *testing.T) {
		perms := domain.RoleCashier.Permissions()
		perms[0] = domain.PermTenantManage
		require.False(t, domain.RoleCashier.Can(domain.PermTenantManage))
	})
}
