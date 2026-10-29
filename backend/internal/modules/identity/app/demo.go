package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// DemoTenantInput is a demonstration tenant: its business, its owner and
// the staff it shows each role with. Every user shares Password.
type DemoTenantInput struct {
	TenantName       string
	NIT              string
	BusinessTemplate string
	Password         string
	Owner            DemoUser
	Staff            []DemoUser
}

// DemoUser is a user of a demo tenant; the owner's Role is ignored.
type DemoUser struct {
	Name  string
	Email string
	Role  domain.Role
}

// SetUpDemoTenant signs up a demo tenant as SignUp does (its template's
// data, in one transaction) with one user for each of its staff. Only the
// demo generator calls it: the contract has no way to mark a tenant demo or
// to add users yet.
func (s *Service) SetUpDemoTenant(ctx context.Context, in DemoTenantInput) (_ uuid.UUID, err error) {
	ctx, span := s.startSpan(ctx, "SetUpDemoTenant")
	defer func() { usecase.End(span, err) }()

	signUp, err := s.parseSignUp(SignUpInput{
		NIT: in.NIT, Email: in.Owner.Email, Password: in.Password, BusinessTemplate: in.BusinessTemplate,
	})
	if err != nil {
		return uuid.Nil, err
	}
	now := s.Now()
	tenant, owner, err := s.newTenantWithOwner(SignUpInput{
		TenantName: in.TenantName, OwnerName: in.Owner.Name, Password: in.Password,
	}, signUp, now)
	if err != nil {
		return uuid.Nil, err
	}
	tenant.MarkDemo()
	staff, err := s.demoStaff(tenant.ID(), owner.PasswordHash(), in.Staff)
	if err != nil {
		return uuid.Nil, err
	}
	span.SetAttributes(attribute.String("tenant.id", tenant.ID().String()),
		attribute.String("business_template", in.BusinessTemplate))

	setUp := func(ctx context.Context) error {
		if err := s.Setup.SetUpTenant(ctx, tenant.ID(), in.BusinessTemplate); err != nil {
			return err
		}
		for _, u := range staff {
			if err := s.Users.Create(ctx, u); err != nil {
				return err
			}
		}
		return nil
	}
	if err := s.Tenants.CreateWithOwner(ctx, tenant, owner, setUp); err != nil {
		return uuid.Nil, err
	}
	s.Logger.InfoContext(ctx, "demo tenant set up",
		slog.String("tenant_id", tenant.ID().String()), slog.String("business_template", in.BusinessTemplate))
	return tenant.ID(), nil
}

// demoStaff builds the staff users with the owner's password hash; none of
// them can be another owner.
func (s *Service) demoStaff(tenantID uuid.UUID, hash string, staff []DemoUser) ([]*domain.User, error) {
	users := make([]*domain.User, 0, len(staff))
	for _, in := range staff {
		if in.Role == domain.RoleOwner {
			return nil, domain.ErrInvalidRole
		}
		email, err := domain.ParseEmail(in.Email)
		if err != nil {
			return nil, err
		}
		u, err := domain.NewUser(tenantID, email, in.Name, hash, in.Role, s.Now())
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// DemoTenants returns the ids of the demo tenants of a business template.
func (s *Service) DemoTenants(ctx context.Context, template string) (_ []uuid.UUID, err error) {
	ctx, span := s.startSpan(ctx, "DemoTenants")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(attribute.String("business_template", template))
	return s.Demo.FindByTemplate(ctx, template)
}

// PurgeDemoTenant deletes a demo tenant with all its data. It refuses any
// other tenant here and again in the database (ErrNotDemoTenant).
func (s *Service) PurgeDemoTenant(ctx context.Context, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "PurgeDemoTenant")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", id.String()))

	tenant, err := s.Tenants.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !tenant.IsDemo() {
		return domain.ErrNotDemoTenant
	}
	if err := s.Demo.Purge(ctx, id); err != nil {
		return err
	}
	s.Logger.InfoContext(ctx, "demo tenant purged", slog.String("tenant_id", id.String()))
	return nil
}
