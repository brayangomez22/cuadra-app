package app

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// SignUpInput registers a business and its owner.
type SignUpInput struct {
	TenantName string
	NIT        string
	OwnerName  string
	Email      string
	Password   string
	// BusinessTemplate is the ID of the template the tenant starts from.
	BusinessTemplate string
}

// SignUp creates the tenant and its owner, and prepares the tenant's initial
// data in the other modules from its business template (Setup), in one
// transaction; then it starts a session. Every field is validated before
// anything is stored or hashed.
func (s *Service) SignUp(ctx context.Context, in SignUpInput) (_ Session, err error) {
	ctx, span := s.startSpan(ctx, "SignUp")
	defer func() { usecase.End(span, err) }()

	signUp, err := s.parseSignUp(in)
	if err != nil {
		return Session{}, err
	}
	template := in.BusinessTemplate
	span.SetAttributes(attribute.String("business_template", template))
	tenant, owner, err := s.newTenantWithOwner(in, signUp, s.Now())
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

// signUpData is what SignUp parses from its input before building anything.
type signUpData struct {
	nit      domain.NIT
	email    domain.Email
	template BusinessTemplate
}

// parseSignUp validates the NIT, the email, the password and the business
// template of in.
func (s *Service) parseSignUp(in SignUpInput) (signUpData, error) {
	nit, err := domain.ParseNIT(in.NIT)
	if err != nil {
		return signUpData{}, err
	}
	email, err := domain.ParseEmail(in.Email)
	if err != nil {
		return signUpData{}, err
	}
	if err := domain.ValidatePassword(in.Password); err != nil {
		return signUpData{}, err
	}
	tpl, ok := s.findTemplate(in.BusinessTemplate)
	if !ok {
		return signUpData{}, domain.ErrUnknownBusinessTemplate
	}
	return signUpData{nit: nit, email: email, template: tpl}, nil
}

// newTenantWithOwner builds the tenant and its owner, with the password
// hashed.
func (s *Service) newTenantWithOwner(in SignUpInput, d signUpData, now time.Time) (*domain.Tenant, *domain.User, error) {
	tenant, err := domain.NewTenant(in.TenantName, d.nit, d.template.ID, d.template.DefaultTaxRate, now)
	if err != nil {
		return nil, nil, err
	}
	hash, err := s.Hasher.Hash(in.Password)
	if err != nil {
		return nil, nil, err
	}
	owner, err := domain.NewUser(tenant.ID(), d.email, in.OwnerName, hash, domain.RoleOwner, now)
	if err != nil {
		return nil, nil, err
	}
	return tenant, owner, nil
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

// BusinessTemplates lists the templates a business can sign up with. They
// are static data, so it opens no span.
func (s *Service) BusinessTemplates() []BusinessTemplate { return s.Setup.Templates() }
