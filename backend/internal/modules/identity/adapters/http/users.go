package httpadapter

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http/identityapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// userNotFoundMessage is the answer for a user the tenant does not have.
const userNotFoundMessage = "El usuario no existe."

// UserUseCases manage the tenant's users.
type UserUseCases interface {
	ListUsers(ctx context.Context, a app.Actor) ([]*domain.User, error)
	ChangeUserRole(ctx context.Context, a app.Actor, userID uuid.UUID, role string) (*domain.User, error)
	DeactivateUser(ctx context.Context, a app.Actor, userID uuid.UUID) (*domain.User, error)
	ActivateUser(ctx context.Context, a app.Actor, userID uuid.UUID) (*domain.User, error)
	IssuePasswordReset(ctx context.Context, a app.Actor, userID uuid.UUID) (*domain.PasswordReset, string, error)
}

// actorFrom returns the authenticated caller; without one the use case
// must not run (domain.ErrUserNotFound answers 401).
func actorFrom(ctx context.Context) (app.Actor, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return app.Actor{}, domain.ErrUserNotFound
	}
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}, nil
}

// targetUser runs a use case on another user: one the tenant does not have
// is 404, not the 401 of a caller that no longer exists.
func targetUser(ctx context.Context, run func(app.Actor) (*domain.User, error)) (*identityapi.User, bool, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, false, err
	}
	u, err := run(a)
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	body := userBody(u)
	return &body, true, nil
}

// ListUsers lists the tenant's users.
func (h *Handler) ListUsers(ctx context.Context, _ identityapi.ListUsersRequestObject) (identityapi.ListUsersResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	users, err := h.uc.ListUsers(ctx, a)
	if err != nil {
		return nil, err
	}
	out := make(identityapi.ListUsers200JSONResponse, 0, len(users))
	for _, u := range users {
		out = append(out, userBody(u))
	}
	return out, nil
}

// UpdateUser changes a user's role.
func (h *Handler) UpdateUser(ctx context.Context, req identityapi.UpdateUserRequestObject) (identityapi.UpdateUserResponseObject, error) {
	body, ok, err := targetUser(ctx, func(a app.Actor) (*domain.User, error) {
		return h.uc.ChangeUserRole(ctx, a, req.UserId, string(req.Body.Role))
	})
	if err != nil || !ok {
		return identityapi.UpdateUser404JSONResponse{NotFoundJSONResponse: notFound()}, err
	}
	return identityapi.UpdateUser200JSONResponse(*body), nil
}

// DeactivateUser blocks a user.
func (h *Handler) DeactivateUser(ctx context.Context, req identityapi.DeactivateUserRequestObject) (identityapi.DeactivateUserResponseObject, error) {
	body, ok, err := targetUser(ctx, func(a app.Actor) (*domain.User, error) {
		return h.uc.DeactivateUser(ctx, a, req.UserId)
	})
	if err != nil || !ok {
		return identityapi.DeactivateUser404JSONResponse{NotFoundJSONResponse: notFound()}, err
	}
	return identityapi.DeactivateUser200JSONResponse(*body), nil
}

// ActivateUser reactivates a user.
func (h *Handler) ActivateUser(ctx context.Context, req identityapi.ActivateUserRequestObject) (identityapi.ActivateUserResponseObject, error) {
	body, ok, err := targetUser(ctx, func(a app.Actor) (*domain.User, error) {
		return h.uc.ActivateUser(ctx, a, req.UserId)
	})
	if err != nil || !ok {
		return identityapi.ActivateUser404JSONResponse{NotFoundJSONResponse: notFound()}, err
	}
	return identityapi.ActivateUser200JSONResponse(*body), nil
}

// IssuePasswordReset issues a reset link for a user.
func (h *Handler) IssuePasswordReset(ctx context.Context, req identityapi.IssuePasswordResetRequestObject) (identityapi.IssuePasswordResetResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	reset, token, err := h.uc.IssuePasswordReset(ctx, a, req.UserId)
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return identityapi.IssuePasswordReset404JSONResponse{NotFoundJSONResponse: notFound()}, nil
	case err != nil:
		return nil, err
	}
	return identityapi.IssuePasswordReset201JSONResponse{Token: token, ExpiresAt: reset.ExpiresAt()}, nil
}

func notFound() identityapi.NotFoundJSONResponse {
	return identityapi.NotFoundJSONResponse(errorBody("not_found", userNotFoundMessage))
}

func userBody(u *domain.User) identityapi.User {
	return identityapi.User{
		Id: u.ID(), Name: u.Name(), Email: u.Email().String(), Role: identityapi.UserRole(u.Role()),
		Active: u.IsActive(), CreatedAt: u.CreatedAt(),
	}
}
