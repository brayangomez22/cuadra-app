// Package app holds the identity use cases: sign-up, login, session refresh
// and logout, and the current user's profile.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Tenants   domain.TenantRepository
	Users     domain.UserRepository
	Tokens    domain.RefreshTokenRepository
	Directory domain.LoginDirectory
	Hasher    domain.PasswordHasher
	Issuer    AccessTokenIssuer
	Setup     TenantSetup
	// Permissions is the role → permission matrix; Modules, the active
	// modules of each tenant.
	Permissions *domain.PermissionMatrix
	Modules     ActiveModules
	// IPLimiter is keyed by client IP and EmailLimiter by normalized email;
	// both count every login attempt.
	IPLimiter      RateLimiter
	EmailLimiter   RateLimiter
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the identity use cases.
type Service struct {
	Deps
	spans usecase.Spans
	// dummyHash is verified when a login has no user to check, so an unknown
	// email takes as long as a wrong password.
	dummyHash string

	logins      metric.Int64Counter
	signups     metric.Int64Counter
	reuseAlerts metric.Int64Counter
}

// NewService builds a Service. It fails if a required dependency is missing
// or a metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Tenants, d.Users, d.Tokens, d.Directory, d.Hasher, d.Issuer, d.Setup,
		d.Permissions, d.Modules, d.IPLimiter, d.EmailLimiter) {
		return nil, errors.New("identity: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)

	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "identity")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.logins, "cuadra.identity.logins",
			"Login attempts, by result (success, failure).", "{attempt}"),
		usecase.Counter(&s.signups, "cuadra.identity.signups",
			"Businesses (tenants) registered, by business template.", "{tenant}"),
		usecase.Counter(&s.reuseAlerts, "cuadra.identity.refresh_reuse_detected",
			"Reused refresh tokens that revoked a live session: a sign of token theft.", "{event}"),
	)
	if err != nil {
		return nil, err
	}
	if s.dummyHash, err = d.Hasher.Hash("cuadra-timing-equalizer"); err != nil {
		return nil, fmt.Errorf("identity: dummy hash: %w", err)
	}
	return s, nil
}

// startSpan opens the use case's span; usecase.End records err on it.
func (s *Service) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return s.spans.Start(ctx, name)
}
