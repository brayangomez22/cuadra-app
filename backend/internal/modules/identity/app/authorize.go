package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// AuthorizationInput are the credentials of a user of TenantID who
// authorizes, on someone else's session, an action that needs Permission
// (a sale over a customer's credit).
type AuthorizationInput struct {
	TenantID   uuid.UUID
	Email      string
	Password   string
	Permission domain.Permission
}

// AuthorizeAction returns the active user of in.TenantID with the
// credentials, if its role grants in.Permission. Like Login, it never tells
// whether the email exists (domain.ErrInvalidCredentials for an unknown
// email, a wrong password or an inactive user) and shares its per-email
// limit, so it cannot be used to guess passwords faster. A valid user whose
// role lacks the permission is domain.ErrNotAuthorized.
func (s *Service) AuthorizeAction(ctx context.Context, in AuthorizationInput) (_ uuid.UUID, err error) {
	ctx, span := s.startSpan(ctx, "AuthorizeAction")
	span.SetAttributes(attribute.String("tenant.id", in.TenantID.String()),
		attribute.String("authorization.permission", string(in.Permission)))
	defer func() {
		result := "granted"
		if err != nil {
			result = failureReason(err)
		}
		s.authorizations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
		usecase.End(span, err)
	}()

	if ok, retryAfter := s.EmailLimiter.Allow(normalizeEmailKey(in.Email)); !ok {
		return uuid.Nil, &RateLimitError{RetryAfter: retryAfter}
	}
	candidates, err := s.loginCandidates(ctx, LoginInput{Email: in.Email, TenantID: in.TenantID})
	if err != nil {
		return uuid.Nil, err
	}
	if len(candidates) == 0 {
		return uuid.Nil, s.rejectWithoutUser(in.Password)
	}
	matches, err := s.matchingUsers(ctx, candidates, in.Password)
	if err != nil {
		return uuid.Nil, err
	}
	if len(matches) != 1 {
		return uuid.Nil, domain.ErrInvalidCredentials
	}
	user := matches[0]
	span.SetAttributes(attribute.String("user.id", user.ID().String()))
	if !s.Permissions.Can(user.Role(), in.Permission) {
		return uuid.Nil, domain.ErrNotAuthorized
	}
	return user.ID(), nil
}
