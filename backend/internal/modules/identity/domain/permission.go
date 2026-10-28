package domain

import (
	"fmt"
	"regexp"
	"slices"
)

// Permission is an action a role may perform, named "<namespace>:<action>"
// (catalog:write, lots:manage). The core declares its own; each activable
// module declares its own when it is registered.
type Permission string

// Permissions of the core, checked by the HTTP middleware RequirePermission.
const (
	PermCatalogRead          Permission = "catalog:read"
	PermCatalogWrite         Permission = "catalog:write"
	PermCustomersRead        Permission = "customers:read"
	PermCustomersWrite       Permission = "customers:write"  // create and edit customers, also from the POS
	PermCustomersManage      Permission = "customers:manage" // the tenant's customer segments
	PermImportsRun           Permission = "imports:run"      // load the catalog and the stock from a file
	PermInventoryRead        Permission = "inventory:read"
	PermInventoryAdjust      Permission = "inventory:adjust"
	PermInventoryCount       Permission = "inventory:count"        // record what is counted in a stock count
	PermInventoryCountManage Permission = "inventory:count_manage" // start, review, approve or cancel stock counts
	PermSalesCreate          Permission = "sales:create"
	PermSalesDiscount        Permission = "sales:discount" // discount without the tenant's limit
	PermSalesRead            Permission = "sales:read"
	PermSalesVoid            Permission = "sales:void"
	PermUsersManage          Permission = "users:manage"
	PermTenantManage         Permission = "tenant:manage"
)

// permissionFormat is "<namespace>:<action>": lowercase letters, digits and
// underscores, each starting with a letter.
var permissionFormat = regexp.MustCompile(`^[a-z][a-z0-9_]*:[a-z][a-z0-9_]*$`)

// ParsePermission returns the permission named s. It checks the format
// only; whether the permission is registered is the PermissionMatrix's to
// say.
func ParsePermission(s string) (Permission, error) {
	if !permissionFormat.MatchString(s) {
		return "", fmt.Errorf("%w: %q", ErrInvalidPermission, s)
	}
	return Permission(s), nil
}

// Grant gives a permission to roles. Module is the activable module that
// declares it, or "" for the core. The owner gets every permission, so it
// need not be listed.
type Grant struct {
	Permission Permission
	Module     string
	Roles      []Role
}

// coreRoles is the core's role → permissions matrix, in the order the
// permissions are listed.
var coreRoles = []struct {
	role        Role
	permissions []Permission
}{
	{RoleAdmin, []Permission{
		PermCatalogRead, PermCatalogWrite, PermImportsRun,
		PermCustomersRead, PermCustomersWrite, PermCustomersManage,
		PermInventoryRead, PermInventoryAdjust, PermInventoryCount, PermInventoryCountManage,
		PermSalesCreate, PermSalesDiscount, PermSalesRead, PermSalesVoid,
		PermUsersManage,
	}},
	{RoleCashier, []Permission{
		PermCatalogRead,
		PermCustomersRead, PermCustomersWrite,
		PermInventoryRead, PermInventoryCount,
		PermSalesCreate, PermSalesRead,
	}},
	{RoleWarehouse, []Permission{
		PermCatalogRead,
		PermCustomersRead,
		PermInventoryRead, PermInventoryAdjust, PermInventoryCount,
	}},
}

// corePermissions lists every core permission. A new one must be added here
// and, for roles other than the owner, to coreRoles.
var corePermissions = []Permission{
	PermCatalogRead,
	PermCatalogWrite,
	PermCustomersRead,
	PermCustomersWrite,
	PermCustomersManage,
	PermImportsRun,
	PermInventoryRead,
	PermInventoryAdjust,
	PermInventoryCount,
	PermInventoryCountManage,
	PermSalesCreate,
	PermSalesDiscount,
	PermSalesRead,
	PermSalesVoid,
	PermUsersManage,
	PermTenantManage,
}

// CoreGrants returns the core permissions, each with the roles other than
// the owner that get it.
func CoreGrants() []Grant {
	grants := make([]Grant, 0, len(corePermissions))
	for _, p := range corePermissions {
		g := Grant{Permission: p}
		for _, cr := range coreRoles {
			if slices.Contains(cr.permissions, p) {
				g.Roles = append(g.Roles, cr.role)
			}
		}
		grants = append(grants, g)
	}
	return grants
}

// PermissionMatrix is the role → permissions matrix of the core and the
// registered modules. It is built once, at startup, and never changes.
type PermissionMatrix struct {
	// grants keeps the registration order, which is the order of the lists.
	grants []Grant
	index  map[Permission]int
}

// NewPermissionMatrix builds the matrix from grants. It fails, so the API
// does not start, if a permission is malformed or registered twice, or if a
// grant names an unknown role.
func NewPermissionMatrix(grants []Grant) (*PermissionMatrix, error) {
	m := &PermissionMatrix{grants: make([]Grant, 0, len(grants)), index: make(map[Permission]int, len(grants))}
	for _, g := range grants {
		if _, err := ParsePermission(string(g.Permission)); err != nil {
			return nil, err
		}
		if _, dup := m.index[g.Permission]; dup {
			return nil, fmt.Errorf("%w: %q", ErrDuplicatePermission, g.Permission)
		}
		for _, r := range g.Roles {
			if !r.valid() {
				return nil, fmt.Errorf("%w: %q in permission %q", ErrInvalidRole, r, g.Permission)
			}
		}
		g.Roles = slices.Clone(g.Roles)
		m.index[g.Permission] = len(m.grants)
		m.grants = append(m.grants, g)
	}
	return m, nil
}

// Known reports whether p is registered.
func (m *PermissionMatrix) Known(p Permission) bool {
	_, ok := m.index[p]
	return ok
}

// Can reports whether r grants p. The owner gets every registered
// permission; an unknown role or permission grants nothing. Whether p's
// module is active is not checked here: its routes answer 404 when it is not.
func (m *PermissionMatrix) Can(r Role, p Permission) bool {
	i, ok := m.index[p]
	if !ok || !r.valid() {
		return false
	}
	return r == RoleOwner || slices.Contains(m.grants[i].Roles, r)
}

// Permissions returns what r grants: the core permissions and those of the
// active modules, in registration order.
func (m *PermissionMatrix) Permissions(r Role, activeModules []string) []Permission {
	var out []Permission
	for _, g := range m.grants {
		if g.Module != "" && !slices.Contains(activeModules, g.Module) {
			continue
		}
		if m.Can(r, g.Permission) {
			out = append(out, g.Permission)
		}
	}
	return out
}
