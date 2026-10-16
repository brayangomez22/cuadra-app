package domain

// Role is a user's role within a tenant. Its permissions come from the
// PermissionMatrix.
type Role string

// The four roles of a tenant.
const (
	RoleOwner     Role = "owner"     // dueño
	RoleAdmin     Role = "admin"     // administrador
	RoleCashier   Role = "cashier"   // cajero
	RoleWarehouse Role = "warehouse" // bodeguero
)

// ParseRole returns the role named s. Matching is exact (lowercase).
func ParseRole(s string) (Role, error) {
	r := Role(s)
	if !r.valid() {
		return "", ErrInvalidRole
	}
	return r, nil
}

func (r Role) valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleCashier, RoleWarehouse:
		return true
	}
	return false
}

func (r Role) String() string { return string(r) }
