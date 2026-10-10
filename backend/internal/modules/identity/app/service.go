// Package app holds the identity use cases: sign-up, login, session refresh
// and logout, and the current user's profile.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"

// AccessTokenIssuer issues short-lived access tokens. platform/auth's JWT
// implements it; the use cases never see the token format.
type AccessTokenIssuer interface {
	Issue(tenantID, userID uuid.UUID, role string) (token string, expiresAt time.Time, err error)
}

// RateLimiter decides whether one more attempt is allowed for key and, when
// it is not, how long to wait.
type RateLimiter interface {
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Tenants   domain.TenantRepository
	Users     domain.UserRepository
	Tokens    domain.RefreshTokenRepository
	Directory domain.LoginDirectory
	Hasher    domain.PasswordHasher
	Issuer    AccessTokenIssuer
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
	tracer trace.Tracer
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
	if d.Tenants == nil || d.Users == nil || d.Tokens == nil || d.Directory == nil ||
		d.Hasher == nil || d.Issuer == nil || d.IPLimiter == nil || d.EmailLimiter == nil {
		return nil, errors.New("identity: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.TracerProvider == nil {
		d.TracerProvider = tracenoop.NewTracerProvider()
	}
	if d.MeterProvider == nil {
		d.MeterProvider = metricnoop.NewMeterProvider()
	}
	if d.Logger == nil {
		d.Logger = slog.New(slog.DiscardHandler)
	}

	s := &Service{Deps: d, tracer: d.TracerProvider.Tracer(instrumentationName)}
	meter := d.MeterProvider.Meter(instrumentationName)
	var err error
	if s.logins, err = meter.Int64Counter("cuadra.auth.logins",
		metric.WithDescription("Login attempts, by result (success, failure)."),
		metric.WithUnit("{attempt}")); err != nil {
		return nil, err
	}
	if s.signups, err = meter.Int64Counter("cuadra.auth.signups",
		metric.WithDescription("Businesses (tenants) registered."),
		metric.WithUnit("{tenant}")); err != nil {
		return nil, err
	}
	if s.reuseAlerts, err = meter.Int64Counter("cuadra.auth.refresh_reuse_detected",
		metric.WithDescription("Reused refresh tokens that revoked a live session: a sign of token theft."),
		metric.WithUnit("{event}")); err != nil {
		return nil, err
	}
	if s.dummyHash, err = d.Hasher.Hash("cuadra-timing-equalizer"); err != nil {
		return nil, fmt.Errorf("identity: dummy hash: %w", err)
	}
	return s, nil
}

// Profile is a user with its tenant.
type Profile struct {
	User   *domain.User
	Tenant *domain.Tenant
}

// Session is what a successful sign-up, login or refresh returns. The
// refresh token is in text form and must reach only the client's cookie.
type Session struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	Profile          Profile
}

// startSession issues an access token and a refresh token that starts a new
// family.
func (s *Service) startSession(ctx context.Context, p Profile) (Session, error) {
	refresh, raw, err := domain.NewRefreshToken(p.Tenant.ID(), p.User.ID(), s.Now())
	if err != nil {
		return Session{}, err
	}
	if err := s.Tokens.Create(ctx, refresh); err != nil {
		return Session{}, err
	}
	return s.session(p, refresh, raw)
}

func (s *Service) session(p Profile, refresh *domain.RefreshToken, raw string) (Session, error) {
	access, expiresAt, err := s.Issuer.Issue(p.Tenant.ID(), p.User.ID(), p.User.Role().String())
	if err != nil {
		return Session{}, fmt.Errorf("identity: issue access token: %w", err)
	}
	return Session{
		AccessToken:      access,
		AccessExpiresAt:  expiresAt,
		RefreshToken:     raw,
		RefreshExpiresAt: refresh.ExpiresAt(),
		Profile:          p,
	}, nil
}

// startSpan opens the use case's span; endSpan records err on it.
func (s *Service) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return s.tracer.Start(ctx, "identity."+name)
}

func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func setProfileAttrs(span trace.Span, p Profile) {
	span.SetAttributes(
		attribute.String("tenant.id", p.Tenant.ID().String()),
		attribute.String("user.id", p.User.ID().String()),
	)
}

// normalizeEmailKey lowercases and trims s for rate limiting, also when it is
// not a valid email.
func normalizeEmailKey(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
