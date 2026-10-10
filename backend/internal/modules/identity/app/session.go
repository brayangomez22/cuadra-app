package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// Refresh exchanges a refresh token for a new session: the token is rotated
// (revoked, and a successor of its family issued) and a new access token is
// issued with the user's current role.
//
// A token presented after it was revoked means two parties hold it, so the
// whole family is revoked and domain.ErrRefreshTokenReused is returned; the
// legitimate client has to log in again. A user that was deactivated, or a
// tenant that was suspended, loses its sessions the same way.
func (s *Service) Refresh(ctx context.Context, raw string) (_ Session, err error) {
	ctx, span := s.startSpan(ctx, "RefreshToken")
	defer func() { endSpan(span, err) }()

	token, err := s.findToken(ctx, raw)
	switch {
	case errors.Is(err, domain.ErrRefreshTokenNotFound):
		return Session{}, domain.ErrInvalidRefreshToken
	case err != nil:
		return Session{}, err
	}
	span.SetAttributes(
		attribute.String("tenant.id", token.TenantID().String()),
		attribute.String("user.id", token.UserID().String()),
	)

	now := s.Now()
	if token.IsRevoked() {
		return Session{}, s.revokeReusedFamily(ctx, token)
	}
	if token.IsExpired(now) {
		return Session{}, domain.ErrRefreshTokenExpired
	}

	profile, err := s.activeProfile(ctx, token.TenantID(), token.UserID())
	if errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrTenantSuspended) {
		if _, err := s.Tokens.RevokeFamily(ctx, token.TenantID(), token.FamilyID(), now); err != nil {
			return Session{}, err
		}
		return Session{}, domain.ErrInvalidRefreshToken
	}
	if err != nil {
		return Session{}, err
	}

	next, nextRaw, err := token.Rotate(now)
	if err != nil {
		return Session{}, err
	}
	if err := s.Tokens.Rotate(ctx, token, next); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenReused) {
			// Another request rotated the same token first.
			return Session{}, s.revokeReusedFamily(ctx, token)
		}
		return Session{}, err
	}
	return s.session(profile, next, nextRaw)
}

// Logout revokes the family of a refresh token. Unknown or malformed tokens
// are ignored: logging out is idempotent. Access tokens already issued stay
// valid until they expire (15 minutes).
func (s *Service) Logout(ctx context.Context, raw string) (err error) {
	ctx, span := s.startSpan(ctx, "Logout")
	defer func() { endSpan(span, err) }()

	token, err := s.findToken(ctx, raw)
	switch {
	case errors.Is(err, domain.ErrRefreshTokenNotFound):
		return nil
	case err != nil:
		return err
	}
	span.SetAttributes(
		attribute.String("tenant.id", token.TenantID().String()),
		attribute.String("user.id", token.UserID().String()),
	)
	_, err = s.Tokens.RevokeFamily(ctx, token.TenantID(), token.FamilyID(), s.Now())
	return err
}

// Me returns the profile of an authenticated user. A user that no longer
// exists or was deactivated is reported as domain.ErrUserNotFound.
func (s *Service) Me(ctx context.Context, tenantID, userID uuid.UUID) (_ Profile, err error) {
	ctx, span := s.startSpan(ctx, "GetMe")
	defer func() { endSpan(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()), attribute.String("user.id", userID.String()))

	profile, err := s.activeProfile(ctx, tenantID, userID)
	if errors.Is(err, domain.ErrTenantSuspended) {
		return Profile{}, domain.ErrUserNotFound
	}
	return profile, err
}

// findToken looks a token up by its text form; a malformed one is reported
// as not found.
func (s *Service) findToken(ctx context.Context, raw string) (*domain.RefreshToken, error) {
	tenantID, hash, err := domain.ParseRefreshToken(raw)
	if err != nil {
		return nil, domain.ErrRefreshTokenNotFound
	}
	return s.Tokens.GetByHash(ctx, tenantID, hash)
}

// activeProfile loads an active user and its active tenant. It returns
// domain.ErrUserNotFound for a missing or inactive user and
// domain.ErrTenantSuspended for a suspended tenant.
func (s *Service) activeProfile(ctx context.Context, tenantID, userID uuid.UUID) (Profile, error) {
	user, err := s.Users.GetByID(ctx, tenantID, userID)
	if err != nil {
		return Profile{}, err
	}
	if !user.IsActive() {
		return Profile{}, domain.ErrUserNotFound
	}
	tenant, err := s.Tenants.GetByID(ctx, tenantID)
	if err != nil {
		return Profile{}, err
	}
	if !tenant.IsActive() {
		return Profile{}, domain.ErrTenantSuspended
	}
	return Profile{User: user, Tenant: tenant}, nil
}

// revokeReusedFamily revokes the family of a reused token. It alerts only
// when that revoked live tokens: presenting a token after logging out (or
// after the family was already revoked) puts no session at risk.
func (s *Service) revokeReusedFamily(ctx context.Context, token *domain.RefreshToken) error {
	revoked, err := s.Tokens.RevokeFamily(ctx, token.TenantID(), token.FamilyID(), s.Now())
	if err != nil {
		return err
	}
	if revoked > 0 {
		s.reuseAlerts.Add(ctx, 1)
		s.Logger.WarnContext(ctx, "refresh token reused: session family revoked",
			slog.String("tenant_id", token.TenantID().String()),
			slog.String("user_id", token.UserID().String()),
			slog.String("family_id", token.FamilyID().String()),
		)
	}
	return domain.ErrRefreshTokenReused
}
