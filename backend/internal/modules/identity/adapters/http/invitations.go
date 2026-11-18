package httpadapter

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http/identityapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// InvitationUseCases invite users and let them join.
type InvitationUseCases interface {
	InviteUser(ctx context.Context, a app.Actor, in app.InvitationInput) (*domain.Invitation, string, error)
	ListInvitations(ctx context.Context, a app.Actor) ([]*domain.Invitation, error)
	RevokeInvitation(ctx context.Context, a app.Actor, id uuid.UUID) error
	InvitationByToken(ctx context.Context, token string) (app.InvitationView, error)
	AcceptInvitation(ctx context.Context, in app.AcceptInvitationInput) (app.Session, error)
}

// ListInvitations lists the open invitations.
func (h *Handler) ListInvitations(ctx context.Context, _ identityapi.ListInvitationsRequestObject) (identityapi.ListInvitationsResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	invs, err := h.uc.ListInvitations(ctx, a)
	if err != nil {
		return nil, err
	}
	out := make(identityapi.ListInvitations200JSONResponse, 0, len(invs))
	for _, inv := range invs {
		out = append(out, h.invitationBody(inv))
	}
	return out, nil
}

// InviteUser invites someone and returns the link's token.
func (h *Handler) InviteUser(ctx context.Context, req identityapi.InviteUserRequestObject) (identityapi.InviteUserResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	inv, token, err := h.uc.InviteUser(ctx, a, app.InvitationInput{
		Email: req.Body.Email, Name: req.Body.Name, Role: string(req.Body.Role),
	})
	if err != nil {
		return nil, err
	}
	return identityapi.InviteUser201JSONResponse{
		Invitation: h.invitationBody(inv),
		Link:       identityapi.IssuedLink{Token: token, ExpiresAt: inv.ExpiresAt()},
	}, nil
}

// RevokeInvitation cancels an invitation.
func (h *Handler) RevokeInvitation(ctx context.Context, req identityapi.RevokeInvitationRequestObject) (identityapi.RevokeInvitationResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.uc.RevokeInvitation(ctx, a, req.InvitationId)
	switch {
	case errors.Is(err, domain.ErrLinkUsed):
		return identityapi.RevokeInvitation409JSONResponse(errorBody("link_used", "La invitación ya fue aceptada o anulada.")), nil
	case err != nil:
		return nil, err
	}
	return identityapi.RevokeInvitation204Response{}, nil
}

// LookupInvitation shows an invitation before it is accepted.
func (h *Handler) LookupInvitation(ctx context.Context, req identityapi.LookupInvitationRequestObject) (identityapi.LookupInvitationResponseObject, error) {
	view, err := h.uc.InvitationByToken(ctx, req.Body.Token)
	if err != nil {
		return nil, err
	}
	inv := view.Invitation
	return identityapi.LookupInvitation200JSONResponse{
		TenantName: view.TenantName, Email: inv.Email().String(), Name: inv.Name(),
		Role: identityapi.UserRole(inv.Role()), ExpiresAt: inv.ExpiresAt(),
	}, nil
}

// AcceptInvitation creates the invitee's user and starts their session.
func (h *Handler) AcceptInvitation(ctx context.Context, req identityapi.AcceptInvitationRequestObject) (identityapi.AcceptInvitationResponseObject, error) {
	session, err := h.uc.AcceptInvitation(ctx, app.AcceptInvitationInput{
		Token: req.Body.Token, Name: req.Body.Name, Password: req.Body.Password,
	})
	if err != nil {
		return nil, err
	}
	body, cookie := h.authSession(session)
	return identityapi.AcceptInvitation200JSONResponse{
		Body: body, Headers: identityapi.AcceptInvitation200ResponseHeaders{SetCookie: &cookie},
	}, nil
}

func (h *Handler) invitationBody(inv *domain.Invitation) identityapi.Invitation {
	return identityapi.Invitation{
		Id: inv.ID(), Email: inv.Email().String(), Name: inv.Name(), Role: identityapi.UserRole(inv.Role()),
		Status: identityapi.InvitationStatus(inv.Status(h.now())), InvitedBy: inv.InvitedBy(),
		ExpiresAt: inv.ExpiresAt(), CreatedAt: inv.CreatedAt(),
	}
}
