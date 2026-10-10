package app

import (
	"context"
	"log/slog"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// SignUpInput registers a business and its owner.
type SignUpInput struct {
	TenantName string
	NIT        string
	OwnerName  string
	Email      string
	Password   string
}

// SignUp creates the tenant and its owner in one transaction and starts a
// session. Every field is validated before anything is stored or hashed.
func (s *Service) SignUp(ctx context.Context, in SignUpInput) (_ Session, err error) {
	ctx, span := s.startSpan(ctx, "SignUp")
	defer func() { endSpan(span, err) }()

	nit, err := domain.ParseNIT(in.NIT)
	if err != nil {
		return Session{}, err
	}
	email, err := domain.ParseEmail(in.Email)
	if err != nil {
		return Session{}, err
	}
	if err := domain.ValidatePassword(in.Password); err != nil {
		return Session{}, err
	}
	now := s.Now()
	tenant, err := domain.NewTenant(in.TenantName, nit, now)
	if err != nil {
		return Session{}, err
	}
	hash, err := s.Hasher.Hash(in.Password)
	if err != nil {
		return Session{}, err
	}
	owner, err := domain.NewUser(tenant.ID(), email, in.OwnerName, hash, domain.RoleOwner, now)
	if err != nil {
		return Session{}, err
	}

	if err := s.Tenants.CreateWithOwner(ctx, tenant, owner); err != nil {
		return Session{}, err
	}
	profile := Profile{User: owner, Tenant: tenant}
	setProfileAttrs(span, profile)
	s.signups.Add(ctx, 1)
	// The request is not authenticated yet: the ids go explicitly.
	s.Logger.InfoContext(ctx, "tenant signed up",
		slog.String("tenant_id", tenant.ID().String()), slog.String("user_id", owner.ID().String()))

	return s.startSession(ctx, profile)
}
