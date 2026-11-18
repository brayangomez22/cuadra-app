package httpadapter_test

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// stubUserUseCases answers the user management, invitation and password
// use cases with whatever each test sets; changeUser answers the role
// change, the deactivation and the activation.
type stubUserUseCases struct {
	listUsers   func(app.Actor) ([]*domain.User, error)
	changeUser  func(a app.Actor, userID uuid.UUID) (*domain.User, error)
	issueReset  func(a app.Actor, userID uuid.UUID) (*domain.PasswordReset, string, error)
	invite      func(app.Actor, app.InvitationInput) (*domain.Invitation, string, error)
	revoke      func(app.Actor, uuid.UUID) error
	lookup      func(token string) (app.InvitationView, error)
	accept      func(app.AcceptInvitationInput) (app.Session, error)
	reset       func(app.ResetPasswordInput) error
	changeOwnPw func(app.Actor, app.PasswordChange) error
}

func (s *stubUserUseCases) ListUsers(_ context.Context, a app.Actor) ([]*domain.User, error) {
	return s.listUsers(a)
}

func (s *stubUserUseCases) ChangeUserRole(_ context.Context, a app.Actor, id uuid.UUID, _ string) (*domain.User, error) {
	return s.changeUser(a, id)
}

func (s *stubUserUseCases) DeactivateUser(_ context.Context, a app.Actor, id uuid.UUID) (*domain.User, error) {
	return s.changeUser(a, id)
}

func (s *stubUserUseCases) ActivateUser(_ context.Context, a app.Actor, id uuid.UUID) (*domain.User, error) {
	return s.changeUser(a, id)
}

func (s *stubUserUseCases) IssuePasswordReset(_ context.Context, a app.Actor, id uuid.UUID) (*domain.PasswordReset, string, error) {
	return s.issueReset(a, id)
}

func (s *stubUserUseCases) InviteUser(_ context.Context, a app.Actor, in app.InvitationInput) (*domain.Invitation, string, error) {
	return s.invite(a, in)
}

func (s *stubUserUseCases) ListInvitations(context.Context, app.Actor) ([]*domain.Invitation, error) {
	return nil, nil
}

func (s *stubUserUseCases) RevokeInvitation(_ context.Context, a app.Actor, id uuid.UUID) error {
	return s.revoke(a, id)
}

func (s *stubUserUseCases) InvitationByToken(_ context.Context, token string) (app.InvitationView, error) {
	return s.lookup(token)
}

func (s *stubUserUseCases) AcceptInvitation(_ context.Context, in app.AcceptInvitationInput) (app.Session, error) {
	return s.accept(in)
}

func (s *stubUserUseCases) ResetPassword(_ context.Context, in app.ResetPasswordInput) error {
	return s.reset(in)
}

func (s *stubUserUseCases) ChangePassword(_ context.Context, a app.Actor, in app.PasswordChange) error {
	return s.changeOwnPw(a, in)
}
