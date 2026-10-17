package app

import (
	"context"
	"log/slog"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// SignUpInput registers a business and its owner.
type SignUpInput struct {
	TenantName string
	NIT        string
	OwnerName  string
	Email      string
	Password   string
	// BusinessTemplate is the ID of the template the tenant starts from;
	// empty means DefaultBusinessTemplate.
	BusinessTemplate string
}

// SignUp creates the tenant and its owner, and prepares the tenant's initial
// data in the other modules from its business template (Setup), in one
// transaction; then it starts a session. Every field is validated before
// anything is stored or hashed.
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
	template := in.BusinessTemplate
	if template == "" {
		template = DefaultBusinessTemplate
	}
	tpl, ok := s.findTemplate(template)
	if !ok {
		return Session{}, domain.ErrUnknownBusinessTemplate
	}
	span.SetAttributes(attribute.String("business_template", template))
	now := s.Now()
	tenant, err := domain.NewTenant(in.TenantName, nit, template, tpl.DefaultTaxRate, now)
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

	setUp := func(ctx context.Context) error { return s.Setup.SetUpTenant(ctx, tenant.ID(), template) }
	if err := s.Tenants.CreateWithOwner(ctx, tenant, owner, setUp); err != nil {
		return Session{}, err
	}
	profile, err := s.profile(ctx, owner, tenant)
	if err != nil {
		return Session{}, err
	}
	setProfileAttrs(span, profile)
	s.signups.Add(ctx, 1, metric.WithAttributes(attribute.String("business_template", template)))
	// The request is not authenticated yet: the ids go explicitly.
	s.Logger.InfoContext(ctx, "tenant signed up",
		slog.String("tenant_id", tenant.ID().String()), slog.String("user_id", owner.ID().String()),
		slog.String("business_template", template))

	return s.startSession(ctx, profile)
}

// findTemplate returns the business template id; ok is false when there is none.
func (s *Service) findTemplate(id string) (_ BusinessTemplate, ok bool) {
	templates := s.Setup.Templates()
	i := slices.IndexFunc(templates, func(t BusinessTemplate) bool { return t.ID == id })
	if i < 0 {
		return BusinessTemplate{}, false
	}
	return templates[i], true
}
