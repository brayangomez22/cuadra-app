package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// LoginInput are the credentials of a login attempt.
type LoginInput struct {
	Email    string
	Password string
	// TenantID picks the tenant when the credentials match several (after a
	// TenantSelectionError); uuid.Nil otherwise. It only narrows the
	// candidates: the password is still checked.
	TenantID uuid.UUID
	// ClientIP keys the per-IP rate limit.
	ClientIP string
}

// TenantOption is a tenant the user can choose to sign in to.
type TenantOption struct {
	ID   uuid.UUID
	Name string
}

// TenantSelectionError means the credentials are valid in several tenants:
// the client must repeat the login with one of them. It matches
// domain.ErrTenantSelectionRequired with errors.Is.
type TenantSelectionError struct {
	Tenants []TenantOption
}

func (e *TenantSelectionError) Error() string {
	return fmt.Sprintf("%v (%d tenants)", domain.ErrTenantSelectionRequired, len(e.Tenants))
}

// Is reports target == domain.ErrTenantSelectionRequired.
func (e *TenantSelectionError) Is(target error) bool {
	return target == domain.ErrTenantSelectionRequired
}

// RateLimitError means too many login attempts; RetryAfter says how long to
// wait. It matches domain.ErrTooManyAttempts with errors.Is.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%v (retry after %s)", domain.ErrTooManyAttempts, e.RetryAfter)
}

// Is reports target == domain.ErrTooManyAttempts.
func (e *RateLimitError) Is(target error) bool { return target == domain.ErrTooManyAttempts }

// Login checks the credentials and starts a session.
//
// It never tells whether the email exists: an unknown email, a wrong
// password and an inactive user all return domain.ErrInvalidCredentials, and
// an unknown email still verifies one hash so it takes as long as a wrong
// password. A suspended tenant is reported (ErrTenantSuspended) only after
// the password matched.
func (s *Service) Login(ctx context.Context, in LoginInput) (_ Session, err error) {
	ctx, span := s.startSpan(ctx, "Login")
	defer func() {
		result := "success"
		if err != nil {
			result = "failure"
			span.SetAttributes(attribute.String("auth.failure_reason", failureReason(err)))
		}
		s.logins.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
		endSpan(span, err)
	}()

	if ok, retryAfter := s.IPLimiter.Allow(in.ClientIP); !ok {
		return Session{}, &RateLimitError{RetryAfter: retryAfter}
	}
	if ok, retryAfter := s.EmailLimiter.Allow(normalizeEmailKey(in.Email)); !ok {
		return Session{}, &RateLimitError{RetryAfter: retryAfter}
	}

	profile, err := s.authenticate(ctx, in)
	if err != nil {
		return Session{}, err
	}
	setProfileAttrs(span, profile)
	return s.startSession(ctx, profile)
}

// authenticate returns the only active user whose password matches among the
// users registered with the email (in in.TenantID, when given).
func (s *Service) authenticate(ctx context.Context, in LoginInput) (Profile, error) {
	email, err := domain.ParseEmail(in.Email)
	if err != nil {
		return Profile{}, s.rejectWithoutUser(in.Password)
	}
	candidates, err := s.Directory.FindByEmail(ctx, email)
	if err != nil {
		return Profile{}, err
	}
	if in.TenantID != uuid.Nil {
		candidates = filterTenant(candidates, in.TenantID)
	}
	if len(candidates) == 0 {
		return Profile{}, s.rejectWithoutUser(in.Password)
	}

	var matches []*domain.User
	for _, c := range candidates {
		user, err := s.Users.GetByID(ctx, c.TenantID, c.UserID)
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			continue // Deleted since the directory lookup.
		case err != nil:
			return Profile{}, err
		}
		// Inactive users are verified too, so they take as long as active ones.
		ok, err := s.Hasher.Verify(in.Password, user.PasswordHash())
		if err != nil {
			return Profile{}, fmt.Errorf("identity: user %s: verify password: %w", user.ID(), err)
		}
		if ok && user.IsActive() {
			matches = append(matches, user)
		}
	}

	switch len(matches) {
	case 0:
		return Profile{}, domain.ErrInvalidCredentials
	case 1:
	default:
		return Profile{}, s.tenantSelection(ctx, matches)
	}

	user := matches[0]
	tenant, err := s.Tenants.GetByID(ctx, user.TenantID())
	if err != nil {
		return Profile{}, err
	}
	if !tenant.IsActive() {
		return Profile{}, domain.ErrTenantSuspended
	}
	return Profile{User: user, Tenant: tenant}, nil
}

// rejectWithoutUser verifies password against a dummy hash, so a login
// without a user to check costs the same as a wrong password.
func (s *Service) rejectWithoutUser(password string) error {
	_, _ = s.Hasher.Verify(password, s.dummyHash)
	return domain.ErrInvalidCredentials
}

func (s *Service) tenantSelection(ctx context.Context, users []*domain.User) error {
	options := make([]TenantOption, 0, len(users))
	for _, u := range users {
		tenant, err := s.Tenants.GetByID(ctx, u.TenantID())
		if err != nil {
			return err
		}
		options = append(options, TenantOption{ID: tenant.ID(), Name: tenant.Name()})
	}
	return &TenantSelectionError{Tenants: options}
}

func filterTenant(candidates []domain.LoginCandidate, tenantID uuid.UUID) []domain.LoginCandidate {
	for _, c := range candidates {
		if c.TenantID == tenantID {
			return []domain.LoginCandidate{c}
		}
	}
	return nil
}

// failureReason is a low-cardinality label of why a login failed, for spans.
func failureReason(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		return "invalid_credentials"
	case errors.Is(err, domain.ErrTooManyAttempts):
		return "rate_limited"
	case errors.Is(err, domain.ErrTenantSuspended):
		return "tenant_suspended"
	case errors.Is(err, domain.ErrTenantSelectionRequired):
		return "tenant_selection_required"
	default:
		return "error"
	}
}
