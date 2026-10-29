package app

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// DemoLoginInput is who enters the demo: a business template and a role.
type DemoLoginInput struct {
	BusinessTemplate string
	Role             string
	// ClientIP keys the per-IP rate limit.
	ClientIP string
}

// StartDemo starts a session, without a password, as the user of in.Role in
// the demo tenant of in.BusinessTemplate. It answers domain.ErrDemoUnavailable
// when the demo is off, when the template has no demo tenant or its user of
// the role is missing, and for any tenant that is not a demo: a session in a
// real business never opens without its password.
func (s *Service) StartDemo(ctx context.Context, in DemoLoginInput) (_ Session, err error) {
	ctx, span := s.startSpan(ctx, "StartDemo")
	defer func() {
		result := "success"
		if err != nil {
			result = "failure"
		}
		s.demoLogins.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result),
			attribute.String("role", in.Role), attribute.String("business_template", in.BusinessTemplate)))
		usecase.End(span, err)
	}()
	span.SetAttributes(attribute.String("business_template", in.BusinessTemplate), attribute.String("role", in.Role))

	if !s.DemoEnabled {
		return Session{}, domain.ErrDemoUnavailable
	}
	if ok, retryAfter := s.DemoLimiter.Allow(in.ClientIP); !ok {
		return Session{}, &RateLimitError{RetryAfter: retryAfter}
	}
	profile, err := s.demoProfile(ctx, in)
	if err != nil {
		return Session{}, err
	}
	setProfileAttrs(span, profile)
	return s.startSession(ctx, profile)
}

// demoProfile is the profile of the demo user of in: the oldest active user
// of the role in the first demo tenant of the template.
func (s *Service) demoProfile(ctx context.Context, in DemoLoginInput) (Profile, error) {
	role, err := domain.ParseRole(in.Role)
	if err != nil {
		return Profile{}, domain.ErrDemoUnavailable
	}
	tenant, err := s.demoTenant(ctx, in.BusinessTemplate)
	if err != nil {
		return Profile{}, err
	}
	user, err := s.Users.FirstByRole(ctx, tenant.ID(), role)
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return Profile{}, domain.ErrDemoUnavailable
	case err != nil:
		return Profile{}, err
	}
	return s.profile(ctx, user, tenant)
}

// demoTenant is the first active demo tenant of template. It checks the
// tenant itself is a demo, not only that the repository listed it.
func (s *Service) demoTenant(ctx context.Context, template string) (*domain.Tenant, error) {
	ids, err := s.Demo.FindByTemplate(ctx, template)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, domain.ErrDemoUnavailable
	}
	tenant, err := s.Tenants.GetByID(ctx, ids[0])
	switch {
	case errors.Is(err, domain.ErrTenantNotFound):
		return nil, domain.ErrDemoUnavailable // Purged by the nightly reset since.
	case err != nil:
		return nil, fmt.Errorf("identity: demo tenant %s: %w", ids[0], err)
	}
	if !tenant.IsDemo() || !tenant.IsActive() {
		return nil, domain.ErrDemoUnavailable
	}
	return tenant, nil
}

// DemoOptions returns the business templates that have a demo tenant, in
// the templates' order; domain.ErrDemoUnavailable when the demo is off.
func (s *Service) DemoOptions(ctx context.Context) (_ []BusinessTemplate, err error) {
	ctx, span := s.startSpan(ctx, "ListDemos")
	defer func() { usecase.End(span, err) }()

	if !s.DemoEnabled {
		return nil, domain.ErrDemoUnavailable
	}
	var out []BusinessTemplate
	for _, t := range s.Setup.Templates() {
		ids, err := s.Demo.FindByTemplate(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		if len(ids) > 0 {
			out = append(out, t)
		}
	}
	return out, nil
}
