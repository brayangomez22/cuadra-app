package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// coreMatrix is the matrix with only the core permissions.
func coreMatrix(t *testing.T) *domain.PermissionMatrix {
	t.Helper()
	m, err := domain.NewPermissionMatrix(domain.CoreGrants())
	require.NoError(t, err)
	return m
}

// moduleMatrix adds the permissions of two activable modules of different
// kinds of business: lot tracking (pharmacies, minimarkets) and work orders
// (repair shops).
func moduleMatrix(t *testing.T) *domain.PermissionMatrix {
	t.Helper()
	grants := append(domain.CoreGrants(),
		domain.Grant{Permission: "lots:manage", Module: "lots_expiry", Roles: []domain.Role{domain.RoleAdmin, domain.RoleWarehouse}},
		domain.Grant{Permission: "work_orders:write", Module: "work_orders", Roles: []domain.Role{domain.RoleCashier}},
	)
	m, err := domain.NewPermissionMatrix(grants)
	require.NoError(t, err)
	return m
}

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

func TestParsePermission(t *testing.T) {
	t.Run("acepta un ID con espacio de nombres", func(t *testing.T) {
		for _, in := range []string{"catalog:write", "lots:manage", "work_orders:write", "sales:void"} {
			got, err := domain.ParsePermission(in)
			require.NoError(t, err, in)
			require.Equal(t, in, string(got))
		}
	})

	t.Run("rechaza un permiso sin espacio de nombres o mal escrito", func(t *testing.T) {
		for _, in := range []string{"", "catalog", ":write", "catalog:", "Catalog:write", "catalog:write:all", "catalog :write", "lots:manage!"} {
			_, err := domain.ParsePermission(in)
			require.ErrorIs(t, err, domain.ErrInvalidPermission, in)
		}
	})
}

func TestPermissionMatrix(t *testing.T) {
	t.Run("los permisos del core conservan su matriz de roles", func(t *testing.T) {
		m := coreMatrix(t)
		tests := []struct {
			name string
			role domain.Role
			perm domain.Permission
			want bool
		}{
			{name: "el cajero no tiene catalog:write", role: domain.RoleCashier, perm: domain.PermCatalogWrite, want: false},
			{name: "el cajero puede vender", role: domain.RoleCashier, perm: domain.PermSalesCreate, want: true},
			{name: "el cajero no tiene sales:discount", role: domain.RoleCashier, perm: domain.PermSalesDiscount, want: false},
			{name: "el admin descuenta sin tope", role: domain.RoleAdmin, perm: domain.PermSalesDiscount, want: true},
			{name: "el bodeguero puede ajustar inventario", role: domain.RoleWarehouse, perm: domain.PermInventoryAdjust, want: true},
			{name: "el bodeguero no puede vender", role: domain.RoleWarehouse, perm: domain.PermSalesCreate, want: false},
			{name: "el cajero ve las ventas", role: domain.RoleCashier, perm: domain.PermSalesRead, want: true},
			{name: "el cajero no puede anular ventas", role: domain.RoleCashier, perm: domain.PermSalesVoid, want: false},
			{name: "el admin anula ventas", role: domain.RoleAdmin, perm: domain.PermSalesVoid, want: true},
			{name: "el dueño anula ventas", role: domain.RoleOwner, perm: domain.PermSalesVoid, want: true},
			{name: "el bodeguero no ve las ventas", role: domain.RoleWarehouse, perm: domain.PermSalesRead, want: false},
			{name: "el admin gestiona usuarios", role: domain.RoleAdmin, perm: domain.PermUsersManage, want: true},
			{name: "el admin no puede gestionar la empresa", role: domain.RoleAdmin, perm: domain.PermTenantManage, want: false},
			{name: "un rol desconocido no tiene permisos", role: domain.Role("root"), perm: domain.PermCatalogRead, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				require.Equal(t, tt.want, m.Can(tt.role, tt.perm))
			})
		}
	})

	t.Run("el owner tiene todos los permisos del core", func(t *testing.T) {
		m := coreMatrix(t)
		var all []domain.Permission
		for _, g := range domain.CoreGrants() {
			all = append(all, g.Permission)
			require.True(t, m.Can(domain.RoleOwner, g.Permission), g.Permission)
		}
		require.Equal(t, all, m.Permissions(domain.RoleOwner, nil))
	})

	t.Run("todos los roles leen catálogo e inventario", func(t *testing.T) {
		m := coreMatrix(t)
		for _, r := range []domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleCashier, domain.RoleWarehouse} {
			require.True(t, m.Can(r, domain.PermCatalogRead), r)
			require.True(t, m.Can(r, domain.PermInventoryRead), r)
		}
	})

	t.Run("registrar dos veces el mismo permiso falla al arrancar", func(t *testing.T) {
		tests := []struct {
			name  string
			extra domain.Grant
		}{
			{name: "un módulo repite un permiso del core", extra: domain.Grant{Permission: domain.PermCatalogWrite, Module: "variants", Roles: []domain.Role{domain.RoleAdmin}}},
			{name: "dos módulos declaran el mismo permiso", extra: domain.Grant{Permission: "lots:manage", Module: "work_orders", Roles: []domain.Role{domain.RoleAdmin}}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				grants := append(domain.CoreGrants(),
					domain.Grant{Permission: "lots:manage", Module: "lots_expiry", Roles: []domain.Role{domain.RoleAdmin}},
					tt.extra)
				_, err := domain.NewPermissionMatrix(grants)
				require.ErrorIs(t, err, domain.ErrDuplicatePermission)
				require.ErrorContains(t, err, string(tt.extra.Permission))
			})
		}
	})

	t.Run("rechaza un permiso mal escrito o un rol desconocido al arrancar", func(t *testing.T) {
		tests := []struct {
			name    string
			grant   domain.Grant
			wantErr error
		}{
			{name: "permiso sin espacio de nombres", grant: domain.Grant{Permission: "manage", Module: "lots_expiry", Roles: []domain.Role{domain.RoleAdmin}}, wantErr: domain.ErrInvalidPermission},
			{name: "rol desconocido", grant: domain.Grant{Permission: "lots:manage", Module: "lots_expiry", Roles: []domain.Role{"root"}}, wantErr: domain.ErrInvalidRole},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewPermissionMatrix(append(domain.CoreGrants(), tt.grant))
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	})

	t.Run("un permiso que no está en el registro se rechaza", func(t *testing.T) {
		m := moduleMatrix(t)
		for _, p := range []domain.Permission{"lots:delete", "reports:read", ""} {
			require.False(t, m.Known(p), p)
			require.False(t, m.Can(domain.RoleOwner, p), p)
		}
		require.True(t, m.Known("lots:manage"))
		require.True(t, m.Known(domain.PermSalesVoid))
	})

	t.Run("un módulo da sus permisos solo a los roles que declara", func(t *testing.T) {
		m := moduleMatrix(t)
		require.True(t, m.Can(domain.RoleAdmin, "lots:manage"))
		require.True(t, m.Can(domain.RoleWarehouse, "lots:manage"))
		require.False(t, m.Can(domain.RoleCashier, "lots:manage"))
		require.True(t, m.Can(domain.RoleCashier, "work_orders:write"))
		require.False(t, m.Can(domain.RoleWarehouse, "work_orders:write"))
	})

	t.Run("el owner recibe todos los permisos de los módulos", func(t *testing.T) {
		m := moduleMatrix(t)
		require.True(t, m.Can(domain.RoleOwner, "lots:manage"))
		require.True(t, m.Can(domain.RoleOwner, "work_orders:write"))
	})

	t.Run("los permisos de un módulo inactivo no aparecen en la lista del rol", func(t *testing.T) {
		m := moduleMatrix(t)

		require.NotContains(t, m.Permissions(domain.RoleAdmin, nil), domain.Permission("lots:manage"))
		require.Contains(t, m.Permissions(domain.RoleAdmin, []string{"lots_expiry"}), domain.Permission("lots:manage"))
		require.NotContains(t, m.Permissions(domain.RoleAdmin, []string{"lots_expiry"}), domain.Permission("work_orders:write"),
			"admin does not get work_orders:write even with the module active")

		cashier := m.Permissions(domain.RoleCashier, []string{"lots_expiry", "work_orders"})
		require.Equal(t, []domain.Permission{
			domain.PermCatalogRead, domain.PermCustomersRead, domain.PermCustomersWrite, domain.PermInventoryRead, domain.PermInventoryCount,
			domain.PermSalesCreate, domain.PermSalesRead, "work_orders:write",
		}, cashier)
	})

	t.Run("modificar la lista de permisos devuelta no altera la matriz", func(t *testing.T) {
		m := coreMatrix(t)
		perms := m.Permissions(domain.RoleCashier, nil)
		perms[0] = domain.PermTenantManage
		require.False(t, m.Can(domain.RoleCashier, domain.PermTenantManage))
	})

	t.Run("modificar los grants después de crear la matriz no la altera", func(t *testing.T) {
		grants := domain.CoreGrants()
		m, err := domain.NewPermissionMatrix(grants)
		require.NoError(t, err)
		for i := range grants {
			grants[i].Roles = append(grants[i].Roles[:0], domain.RoleWarehouse)
		}
		require.False(t, m.Can(domain.RoleWarehouse, domain.PermSalesCreate))
		require.True(t, m.Can(domain.RoleCashier, domain.PermSalesCreate))
	})
}
