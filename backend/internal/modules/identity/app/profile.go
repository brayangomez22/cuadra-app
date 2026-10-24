package app

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// Profile is a user with its tenant.
type Profile struct {
	User   *domain.User
	Tenant *domain.Tenant
	// Modules are the tenant's active modules, and Permissions what the
	// user's role grants with them.
	Modules     []string
	Permissions []domain.Permission
}

func setProfileAttrs(span trace.Span, p Profile) {
	span.SetAttributes(
		attribute.String("tenant.id", p.Tenant.ID().String()),
		attribute.String("user.id", p.User.ID().String()),
	)
}

// NewPermissionMatrix builds the matrix of the core permissions and those
// the registered modules declare. It fails, so the API does not start, if a
// module repeats a permission, misspells it or names an unknown role.
func NewPermissionMatrix(registry ModuleRegistry) (*domain.PermissionMatrix, error) {
	grants := domain.CoreGrants()
	for _, mp := range registry.Permissions() {
		g := domain.Grant{Permission: domain.Permission(mp.Permission), Module: mp.Module}
		for _, name := range mp.Roles {
			role, err := domain.ParseRole(name)
			if err != nil {
				return nil, fmt.Errorf("identity: module %s, permission %s: %w", mp.Module, mp.Permission, err)
			}
			g.Roles = append(g.Roles, role)
		}
		grants = append(grants, g)
	}
	return domain.NewPermissionMatrix(grants)
}

// profile completes a user's profile with the tenant's active modules and
// what the user's role grants with them.
func (s *Service) profile(ctx context.Context, user *domain.User, tenant *domain.Tenant) (Profile, error) {
	modules, err := s.Modules.Active(ctx, tenant.ID())
	if err != nil {
		return Profile{}, fmt.Errorf("identity: active modules: %w", err)
	}
	return Profile{
		User:        user,
		Tenant:      tenant,
		Modules:     modules,
		Permissions: s.Permissions.Permissions(user.Role(), modules),
	}, nil
}
