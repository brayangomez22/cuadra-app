package domain

import "slices"

// Permission is an action a role may perform, named "<module>:<action>".
type Permission string

// Permissions checked by the HTTP middleware RequirePermission.
const (
	PermCatalogRead     Permission = "catalog:read"
	PermCatalogWrite    Permission = "catalog:write"
	PermInventoryRead   Permission = "inventory:read"
	PermInventoryAdjust Permission = "inventory:adjust"
	PermSalesCreate     Permission = "sales:create"
	PermUsersManage     Permission = "users:manage"
	PermTenantManage    Permission = "tenant:manage"
)

// allPermissions lists every permission. A new permission must be added here;
// the owner gets it automatically.
var allPermissions = []Permission{
	PermCatalogRead,
	PermCatalogWrite,
	PermInventoryRead,
	PermInventoryAdjust,
	PermSalesCreate,
	PermUsersManage,
	PermTenantManage,
}

// rolePermissions is the role → permissions matrix. It is also the list of
// valid roles.
var rolePermissions = map[Role][]Permission{
	RoleOwner: allPermissions,
	RoleAdmin: {
		PermCatalogRead, PermCatalogWrite,
		PermInventoryRead, PermInventoryAdjust,
		PermSalesCreate,
		PermUsersManage,
	},
	RoleCashier: {
		PermCatalogRead,
		PermInventoryRead,
		PermSalesCreate,
	},
	RoleWarehouse: {
		PermCatalogRead,
		PermInventoryRead, PermInventoryAdjust,
	},
}

// AllPermissions returns a copy of every defined permission.
func AllPermissions() []Permission { return slices.Clone(allPermissions) }

// Can reports whether the role grants p. Unknown roles grant nothing.
func (r Role) Can(p Permission) bool {
	return slices.Contains(rolePermissions[r], p)
}

// Permissions returns a copy of the permissions granted by the role.
func (r Role) Permissions() []Permission { return slices.Clone(rolePermissions[r]) }
