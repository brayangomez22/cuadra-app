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
	Tenants domain.TenantRepository
	Users   domain.UserRepository
	Tokens  domain.RefreshTokenRepository
	// ExpiredTokens purges the expired refresh tokens of every tenant.
	ExpiredTokens domain.ExpiredRefreshTokens
	// Invitations and Resets are the links to join the tenant and to
	// choose a new password.
	Invitations domain.InvitationRepository
	Resets      domain.PasswordResetRepository
	Directory   domain.LoginDirectory
	Demo        domain.DemoTenantRepository
	Hasher      domain.PasswordHasher
	Issuer      AccessTokenIssuer
	Setup       TenantSetup
	// Permissions is the role → permission matrix; Modules, the active
	// modules of each tenant.
	Permissions *domain.PermissionMatrix
	Modules     ActiveModules
	// Flags are the feature flags of each tenant, for /me; Pseudonyms, the
	// IDs of the users in the usage analytics.
	Flags      Flags
	Pseudonyms Pseudonyms
	// IPLimiter is keyed by client IP and EmailLimiter by normalized email;
	// both count every login attempt.
	IPLimiter    RateLimiter
	EmailLimiter RateLimiter
	// DemoEnabled lets anyone enter the demo tenants without a password
	// (DEMO_ENABLED); DemoLimiter, keyed by client IP, counts those entries.
	DemoEnabled bool
	DemoLimiter RateLimiter
	// SignupLimiter, keyed by client IP, counts the sign-ups.
	SignupLimiter RateLimiter
	// Audit records the changes of the tenant's settings, stored together
	// through Tx.
	Audit          AuditLog
	Tx             Transactor
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

	logins         metric.Int64Counter
	authorizations metric.Int64Counter
	signups        metric.Int64Counter
	reuseAlerts    metric.Int64Counter
	demoLogins     metric.Int64Counter
	tokensPurged   metric.Int64Counter
	userChanges    metric.Int64Counter
	invitations    metric.Int64Counter
	passwordResets metric.Int64Counter
	rejected       metric.Int64Counter
}

// NewService builds a Service. It fails if a required dependency is missing
// or a metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Tenants, d.Users, d.Tokens, d.ExpiredTokens, d.Invitations, d.Resets, d.Directory, d.Demo, d.Hasher, d.Issuer, d.Setup,
		d.Permissions, d.Modules, d.Flags, d.Pseudonyms, d.IPLimiter, d.EmailLimiter, d.DemoLimiter, d.SignupLimiter, d.Audit, d.Tx) {
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
		usecase.Counter(&s.authorizations, "cuadra.identity.authorizations",
			"Authorizations of an action on someone else's session, by result (granted or why not).", "{attempt}"),
		usecase.Counter(&s.signups, "cuadra.identity.signups",
			"Businesses (tenants) registered, by business template.", "{tenant}"),
		usecase.Counter(&s.reuseAlerts, "cuadra.identity.refresh_reuse_detected",
			"Reused refresh tokens that revoked a live session: a sign of token theft.", "{event}"),
		usecase.Counter(&s.demoLogins, "cuadra.identity.demo_logins",
			"Entries to the public demo, by result, role and business template.", "{session}"),
		usecase.Counter(&s.tokensPurged, "cuadra.identity.refresh_tokens_purged",
			"Expired refresh tokens deleted by the nightly purge.", "{token}"),
		usecase.Counter(&s.userChanges, "cuadra.identity.user_changes",
			"Changes to a user by an owner or admin, by action (role_changed, deactivated, activated).", "{change}"),
		usecase.Counter(&s.invitations, "cuadra.identity.invitations",
			"Invitations to join a tenant, by result (sent, accepted, revoked).", "{invitation}"),
		usecase.Counter(&s.passwordResets, "cuadra.identity.password_resets",
			"Password reset links, by result (issued, completed).", "{reset}"),
		usecase.Counter(&s.rejected, "cuadra.identity.sessions_rejected",
			"Requests with a valid access token of a user that is no longer active, by reason.", "{request}"),
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
