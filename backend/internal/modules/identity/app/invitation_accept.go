package app

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// InvitationView is what the invitee sees before accepting: the
// invitation and the name of the company that invites.
type InvitationView struct {
	Invitation *domain.Invitation
	TenantName string
}

// InvitationByToken returns the usable invitation of a link: a malformed
// or unknown token is domain.ErrLinkNotFound, and one that expired or was
// accepted or revoked, domain.ErrLinkExpired or domain.ErrLinkUsed.
func (s *Service) InvitationByToken(ctx context.Context, token string) (_ InvitationView, err error) {
	ctx, span := s.startSpan(ctx, "InvitationByToken")
	defer func() { usecase.End(span, err) }()

	inv, err := s.findInvitation(ctx, token)
	if err != nil {
		return InvitationView{}, err
	}
	span.SetAttributes(attribute.String("tenant.id", inv.TenantID().String()))
	if err := inv.Accept(s.Now()); err != nil { // checks only: inv is not stored
		return InvitationView{}, err
	}
	tenant, err := s.activeTenant(ctx, inv)
	if err != nil {
		return InvitationView{}, err
	}
	return InvitationView{Invitation: inv, TenantName: tenant.Name()}, nil
}

// AcceptInvitationInput is the invitee's answer: the link's token, the
// name they go by and the password they choose.
type AcceptInvitationInput struct {
	Token    string
	Name     string
	Password string
}

// AcceptInvitation creates the invitee's user with the invited role and
// starts their session. The link works once.
func (s *Service) AcceptInvitation(ctx context.Context, in AcceptInvitationInput) (_ Session, err error) {
	ctx, span := s.startSpan(ctx, "AcceptInvitation")
	defer func() { usecase.End(span, err) }()

	if err := domain.ValidatePassword(in.Password); err != nil {
		return Session{}, err
	}
	inv, err := s.findInvitation(ctx, in.Token)
	if err != nil {
		return Session{}, err
	}
	tenant, err := s.activeTenant(ctx, inv)
	if err != nil {
		return Session{}, err
	}
	user, err := s.joinTenant(ctx, inv, in)
	if err != nil {
		return Session{}, err
	}
	s.invitations.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "accepted")))
	profile, err := s.profile(ctx, user, tenant)
	if err != nil {
		return Session{}, err
	}
	setProfileAttrs(span, profile)
	return s.startSession(ctx, profile)
}

// joinTenant accepts inv and creates its user in one transaction.
func (s *Service) joinTenant(ctx context.Context, inv *domain.Invitation, in AcceptInvitationInput) (*domain.User, error) {
	if err := inv.Accept(s.Now()); err != nil {
		return nil, err
	}
	hash, err := s.Hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	user, err := domain.NewUser(inv.TenantID(), inv.Email(), in.Name, hash, inv.Role(), s.Now())
	if err != nil {
		return nil, err
	}
	err = s.Tx.InTx(ctx, inv.TenantID(), func(ctx context.Context) error {
		if err := s.Invitations.Close(ctx, inv); err != nil {
			return err
		}
		return s.Users.Create(ctx, user)
	})
	return user, err
}

// findInvitation looks an invitation up by its link token.
func (s *Service) findInvitation(ctx context.Context, token string) (*domain.Invitation, error) {
	tenantID, hash, err := domain.ParseLinkToken(token)
	if err != nil {
		return nil, domain.ErrLinkNotFound
	}
	return s.Invitations.GetByHash(ctx, tenantID, hash)
}

// activeTenant returns the tenant of inv; a suspended one is
// domain.ErrTenantSuspended.
func (s *Service) activeTenant(ctx context.Context, inv *domain.Invitation) (*domain.Tenant, error) {
	tenant, err := s.Tenants.GetByID(ctx, inv.TenantID())
	if err != nil {
		return nil, err
	}
	if !tenant.IsActive() {
		return nil, domain.ErrTenantSuspended
	}
	return tenant, nil
}
