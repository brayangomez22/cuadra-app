package domain

// Role is a user's role within a tenant. Its permissions come from the
// matrix in permission.go.
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
	_, ok := rolePermissions[r]
	return ok
}

func (r Role) String() string { return string(r) }
