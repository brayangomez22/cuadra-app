package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http/identityapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
)

// PasswordUseCases change passwords.
type PasswordUseCases interface {
	ResetPassword(ctx context.Context, in app.ResetPasswordInput) error
	ChangePassword(ctx context.Context, a app.Actor, in app.PasswordChange) error
}

// ResetPassword sets a new password with a reset link.
func (h *Handler) ResetPassword(ctx context.Context, req identityapi.ResetPasswordRequestObject) (identityapi.ResetPasswordResponseObject, error) {
	err := h.uc.ResetPassword(ctx, app.ResetPasswordInput{Token: req.Body.Token, Password: req.Body.Password})
	if err != nil {
		return nil, err
	}
	return identityapi.ResetPassword204Response{}, nil
}

// ChangeMyPassword changes the caller's own password.
func (h *Handler) ChangeMyPassword(ctx context.Context, req identityapi.ChangeMyPasswordRequestObject) (identityapi.ChangeMyPasswordResponseObject, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return nil, err
	}
	err = h.uc.ChangePassword(ctx, a, app.PasswordChange{Current: req.Body.CurrentPassword, New: req.Body.NewPassword})
	if err != nil {
		return nil, err
	}
	return identityapi.ChangeMyPassword204Response{}, nil
}
