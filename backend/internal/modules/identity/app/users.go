package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Audit actions of managing users (docs/auditoria.md).
const (
	auditUserRoleChanged   = "user.role_changed"
	auditUserDeactivated   = "user.deactivated"
	auditUserActivated     = "user.activated"
	auditUserInvited       = "user.invited"
	auditUserPasswordReset = "user.password_reset"
)

// Actor is the authenticated user who runs a use case. Its role is read
// from the store, not from the token, so a change applies at once.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

func (a Actor) attrs() []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String("tenant.id", a.TenantID.String()), attribute.String("user.id", a.UserID.String())}
}

// ListUsers returns the tenant's users, active or not, by name.
func (s *Service) ListUsers(ctx context.Context, a Actor) (_ []*domain.User, err error) {
	ctx, span := s.startSpan(ctx, "ListUsers")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(a.attrs()...)

	return s.Users.List(ctx, a.TenantID)
}

// ChangeUserRole gives userID the role named role, on behalf of a.
func (s *Service) ChangeUserRole(ctx context.Context, a Actor, userID uuid.UUID, role string) (_ *domain.User, err error) {
	ctx, span := s.startSpan(ctx, "ChangeUserRole")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(append(a.attrs(), attribute.String("target.id", userID.String()))...)

	r, err := domain.ParseRole(role)
	if err != nil {
		return nil, err
	}
	return s.changeUser(ctx, a, userID, "role_changed", func(actor, target *domain.User, owners int) (AuditEntry, error) {
		old := target.Role()
		if err := target.ChangeRoleBy(actor, r, owners); err != nil {
			return AuditEntry{}, err
		}
		return AuditEntry{Action: auditUserRoleChanged, EntityID: userID,
			Details: map[string]string{"old_role": old.String(), "new_role": r.String()}}, nil
	})
}

// DeactivateUser blocks userID on behalf of a and ends all their
// sessions: their next request is rejected.
func (s *Service) DeactivateUser(ctx context.Context, a Actor, userID uuid.UUID) (_ *domain.User, err error) {
	ctx, span := s.startSpan(ctx, "DeactivateUser")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(append(a.attrs(), attribute.String("target.id", userID.String()))...)

	return s.changeUser(ctx, a, userID, "deactivated", func(actor, target *domain.User, owners int) (AuditEntry, error) {
		if err := target.DeactivateBy(actor, owners); err != nil {
			return AuditEntry{}, err
		}
		if _, err := s.Tokens.RevokeUser(ctx, a.TenantID, userID, s.Now()); err != nil {
			return AuditEntry{}, err
		}
		return AuditEntry{Action: auditUserDeactivated, EntityID: userID, Details: map[string]string{}}, nil
	})
}

// ActivateUser reactivates userID on behalf of a.
func (s *Service) ActivateUser(ctx context.Context, a Actor, userID uuid.UUID) (_ *domain.User, err error) {
	ctx, span := s.startSpan(ctx, "ActivateUser")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(append(a.attrs(), attribute.String("target.id", userID.String()))...)

	return s.changeUser(ctx, a, userID, "activated", func(actor, target *domain.User, _ int) (AuditEntry, error) {
		if err := target.ActivateBy(actor); err != nil {
			return AuditEntry{}, err
		}
		return AuditEntry{Action: auditUserActivated, EntityID: userID, Details: map[string]string{}}, nil
	})
}

// userChange applies a change to target on behalf of actor, given the
// tenant's active owners, and returns its audit entry.
type userChange func(actor, target *domain.User, activeOwners int) (AuditEntry, error)

// changeUser runs change in one transaction with the active owners locked,
// then stores the user and records the audit entry.
func (s *Service) changeUser(ctx context.Context, a Actor, userID uuid.UUID, action string, change userChange) (*domain.User, error) {
	var target *domain.User
	err := s.Tx.InTx(ctx, a.TenantID, func(ctx context.Context) error {
		owners, err := s.Users.LockActiveOwners(ctx, a.TenantID)
		if err != nil {
			return err
		}
		actor, err := s.Users.GetByID(ctx, a.TenantID, a.UserID)
		if err != nil {
			return err
		}
		if target, err = s.Users.GetByID(ctx, a.TenantID, userID); err != nil {
			return err
		}
		entry, err := change(actor, target, owners)
		if err != nil {
			return err
		}
		if err := s.Users.Update(ctx, target); err != nil {
			return err
		}
		return s.Audit.Record(ctx, a.TenantID, a.UserID, entry)
	})
	if err != nil {
		return nil, err
	}
	s.userChanges.Add(ctx, 1, metric.WithAttributes(attribute.String("action", action)))
	return target, nil
}

// CurrentRole returns the stored role of an authenticated user, for every
// request: a user that no longer exists or was deactivated is
// domain.ErrUserNotFound, so a valid access token stops working at once.
func (s *Service) CurrentRole(ctx context.Context, tenantID, userID uuid.UUID) (_ domain.Role, err error) {
	ctx, span := s.startSpan(ctx, "CurrentRole")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()), attribute.String("user.id", userID.String()))

	u, err := s.Users.GetByID(ctx, tenantID, userID)
	switch {
	case err == nil && u.IsActive():
		return u.Role(), nil
	case err == nil:
		s.rejected.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", "inactive")))
		return "", domain.ErrUserNotFound
	case errors.Is(err, domain.ErrUserNotFound):
		s.rejected.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", "not_found")))
	}
	return "", err
}
