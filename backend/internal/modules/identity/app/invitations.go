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

// InvitationInput is whom an owner or admin invites, and with which role.
type InvitationInput struct {
	Email string
	Name  string
	Role  string
}

// InviteUser invites someone to the tenant on behalf of a and returns the
// invitation with its link token, which is shown once: the inviter shares
// the link. Inviting an email again revokes its earlier links; an email of
// a user of the tenant is domain.ErrEmailTaken.
func (s *Service) InviteUser(ctx context.Context, a Actor, in InvitationInput) (_ *domain.Invitation, token string, err error) {
	ctx, span := s.startSpan(ctx, "InviteUser")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(a.attrs()...)

	email, role, err := parseInvitation(in)
	if err != nil {
		return nil, "", err
	}
	var inv *domain.Invitation
	err = s.Tx.InTx(ctx, a.TenantID, func(ctx context.Context) error {
		if inv, token, err = s.newInvitation(ctx, a, email, in.Name, role); err != nil {
			return err
		}
		if err := s.Invitations.RevokeOpen(ctx, a.TenantID, email, s.Now()); err != nil {
			return err
		}
		if err := s.Invitations.Create(ctx, inv); err != nil {
			return err
		}
		return s.Audit.Record(ctx, a.TenantID, a.UserID, AuditEntry{
			Action: auditUserInvited, EntityID: inv.ID(), Details: map[string]string{"role": role.String()},
		})
	})
	if err != nil {
		return nil, "", err
	}
	s.invitations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "sent")))
	return inv, token, nil
}

func parseInvitation(in InvitationInput) (domain.Email, domain.Role, error) {
	email, err := domain.ParseEmail(in.Email)
	if err != nil {
		return domain.Email{}, "", err
	}
	role, err := domain.ParseRole(in.Role)
	return email, role, err
}

// newInvitation checks that actor may give role and that email is not a
// user of the tenant yet, and builds the invitation.
func (s *Service) newInvitation(ctx context.Context, a Actor, email domain.Email, name string, role domain.Role) (*domain.Invitation, string, error) {
	actor, err := s.Users.GetByID(ctx, a.TenantID, a.UserID)
	if err != nil {
		return nil, "", err
	}
	if err := domain.CheckAssignable(actor.Role(), role); err != nil {
		return nil, "", err
	}
	switch _, err := s.Users.GetByEmail(ctx, a.TenantID, email); {
	case err == nil:
		return nil, "", domain.ErrEmailTaken
	case !errors.Is(err, domain.ErrUserNotFound):
		return nil, "", err
	}
	return domain.NewInvitation(a.TenantID, email, name, role, a.UserID, s.Now())
}

// ListInvitations returns the tenant's invitations neither accepted nor
// revoked, newest first; their Status tells the expired ones.
func (s *Service) ListInvitations(ctx context.Context, a Actor) (_ []*domain.Invitation, err error) {
	ctx, span := s.startSpan(ctx, "ListInvitations")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(a.attrs()...)

	return s.Invitations.ListOpen(ctx, a.TenantID)
}

// RevokeInvitation cancels an invitation: its link stops working.
func (s *Service) RevokeInvitation(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "RevokeInvitation")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(append(a.attrs(), attribute.String("invitation.id", id.String()))...)

	err = s.Tx.InTx(ctx, a.TenantID, func(ctx context.Context) error {
		inv, err := s.Invitations.GetByID(ctx, a.TenantID, id)
		if err != nil {
			return err
		}
		if err := inv.Revoke(s.Now()); err != nil {
			return err
		}
		return s.Invitations.Close(ctx, inv)
	})
	if err == nil {
		s.invitations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "revoked")))
	}
	return err
}
