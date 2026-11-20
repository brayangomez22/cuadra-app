// Package httpadapter exposes the getting-started checklist over HTTP: it
// implements the strict server generated from the "onboarding" operations
// of api/openapi.yaml (onboardingapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/adapters/http/onboardingapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the onboarding domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrNothingToDismiss, Status: http.StatusConflict, Code: "nothing_to_dismiss",
		Message: "No tiene primeros pasos que ocultar."},
}

// UseCases are the getting-started use cases the handlers call (*app.Service).
type UseCases interface {
	GetChecklist(ctx context.Context, a app.Actor) (domain.Checklist, error)
	DismissChecklist(ctx context.Context, a app.Actor) error
}

// Handler implements onboardingapi.StrictServerInterface.
type Handler struct{ uc UseCases }

var _ onboardingapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the onboarding operations on mux, with their error mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	errs := httpx.NewErrorMapper(log, errorRules...)
	onboardingapi.HandlerWithOptions(
		onboardingapi.NewStrictHandlerWithOptions(&Handler{uc: uc}, []onboardingapi.StrictMiddlewareFunc{auth.RequirePrincipal[onboardingapi.StrictHandlerFunc]}, onboardingapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		onboardingapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor is the token's user.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID, Role: p.Role}
}

// GetOnboardingChecklist returns the token's user's checklist.
func (h *Handler) GetOnboardingChecklist(ctx context.Context, _ onboardingapi.GetOnboardingChecklistRequestObject) (onboardingapi.GetOnboardingChecklistResponseObject, error) {
	c, err := h.uc.GetChecklist(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	steps := make([]onboardingapi.OnboardingStep, 0, len(c.Steps))
	for _, s := range c.Steps {
		steps = append(steps, onboardingapi.OnboardingStep{
			Id: onboardingapi.OnboardingStepId(s.ID), Done: s.Done, Critical: s.Critical,
		})
	}
	return onboardingapi.GetOnboardingChecklist200JSONResponse{
		Visible: c.Visible(), Dismissed: c.Dismissed, Completed: c.Completed(), Total: c.Total(), Steps: steps,
	}, nil
}

// DismissOnboardingChecklist hides the token's user's checklist.
func (h *Handler) DismissOnboardingChecklist(ctx context.Context, _ onboardingapi.DismissOnboardingChecklistRequestObject) (onboardingapi.DismissOnboardingChecklistResponseObject, error) {
	if err := h.uc.DismissChecklist(ctx, actor(ctx)); err != nil {
		return nil, err
	}
	return onboardingapi.DismissOnboardingChecklist204Response{}, nil
}
