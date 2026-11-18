package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// IssuePasswordReset issues, on behalf of a, a link for userID to choose a
// new password, and returns it with its token, which is shown once. Only
// the last link issued for a user works.
func (s *Service) IssuePasswordReset(ctx context.Context, a Actor, userID uuid.UUID) (_ *domain.PasswordReset, token string, err error) {
	ctx, span := s.startSpan(ctx, "IssuePasswordReset")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(append(a.attrs(), attribute.String("target.id", userID.String()))...)

	var reset *domain.PasswordReset
	err = s.Tx.InTx(ctx, a.TenantID, func(ctx context.Context) error {
		if err := s.checkManages(ctx, a, userID); err != nil {
			return err
		}
		if reset, token, err = domain.NewPasswordReset(a.TenantID, userID, a.UserID, s.Now()); err != nil {
			return err
		}
		if err := s.Resets.Issue(ctx, reset); err != nil {
			return err
		}
		return s.Audit.Record(ctx, a.TenantID, a.UserID, AuditEntry{
			Action: auditUserPasswordReset, EntityID: userID, Details: map[string]string{},
		})
	})
	if err != nil {
		return nil, "", err
	}
	s.passwordResets.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "issued")))
	return reset, token, nil
}

// checkManages checks that the actor may change the user userID.
func (s *Service) checkManages(ctx context.Context, a Actor, userID uuid.UUID) error {
	actor, err := s.Users.GetByID(ctx, a.TenantID, a.UserID)
	if err != nil {
		return err
	}
	target, err := s.Users.GetByID(ctx, a.TenantID, userID)
	if err != nil {
		return err
	}
	return domain.CheckManages(actor, target)
}

// ResetPasswordInput is the token of a reset link and the new password.
type ResetPasswordInput struct {
	Token    string
	Password string
}

// ResetPassword sets the new password of a reset link's user and ends all
// their sessions. The link works once.
func (s *Service) ResetPassword(ctx context.Context, in ResetPasswordInput) (err error) {
	ctx, span := s.startSpan(ctx, "ResetPassword")
	defer func() { usecase.End(span, err) }()

	if err := domain.ValidatePassword(in.Password); err != nil {
		return err
	}
	tenantID, hash, err := domain.ParseLinkToken(in.Token)
	if err != nil {
		return domain.ErrLinkNotFound
	}
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()))
	passwordHash, err := s.Hasher.Hash(in.Password)
	if err != nil {
		return err
	}
	err = s.Tx.InTx(ctx, tenantID, func(ctx context.Context) error {
		reset, err := s.Resets.GetByHash(ctx, tenantID, hash)
		if err != nil {
			return err
		}
		if err := reset.Use(s.Now()); err != nil {
			return err
		}
		if err := s.Resets.MarkUsed(ctx, reset); err != nil {
			return err
		}
		return s.setPassword(ctx, tenantID, reset.UserID(), passwordHash)
	})
	if err == nil {
		s.passwordResets.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "completed")))
	}
	return err
}

// setPassword stores the user's new password hash and ends their sessions.
func (s *Service) setPassword(ctx context.Context, tenantID, userID uuid.UUID, hash string) error {
	user, err := s.Users.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if err := user.SetPasswordHash(hash); err != nil {
		return err
	}
	if err := s.Users.Update(ctx, user); err != nil {
		return err
	}
	_, err = s.Tokens.RevokeUser(ctx, tenantID, userID, s.Now())
	return err
}

// PasswordChange is a user's current password and the one they choose.
type PasswordChange struct {
	Current string
	New     string
}

// ChangePassword changes the actor's own password; the current one must
// match (domain.ErrWrongPassword). Their sessions stay open: they are the
// one who changes it.
func (s *Service) ChangePassword(ctx context.Context, a Actor, in PasswordChange) (err error) {
	ctx, span := s.startSpan(ctx, "ChangePassword")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(a.attrs()...)

	if err := domain.ValidatePassword(in.New); err != nil {
		return err
	}
	user, err := s.Users.GetByID(ctx, a.TenantID, a.UserID)
	if err != nil {
		return err
	}
	ok, err := s.Hasher.Verify(in.Current, user.PasswordHash())
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrWrongPassword
	}
	hash, err := s.Hasher.Hash(in.New)
	if err != nil {
		return err
	}
	if err := user.SetPasswordHash(hash); err != nil {
		return err
	}
	return s.Users.Update(ctx, user)
}
