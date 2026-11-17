// Package httpadapter exposes the feedback over HTTP: it implements the
// strict server generated from the "feedback" operations of
// api/openapi.yaml (feedbackapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/http/feedbackapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the feedback domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrInvalidMessage, Status: http.StatusUnprocessableEntity, Code: "invalid_feedback",
		Message: "Escriba su comentario (hasta 2000 caracteres)."},
	{Err: domain.ErrInvalidScreen, Status: http.StatusUnprocessableEntity, Code: "invalid_screen",
		Message: "La pantalla del comentario no es válida."},
	{Err: domain.ErrInvalidVersion, Status: http.StatusUnprocessableEntity, Code: "invalid_app_version",
		Message: "La versión de la app no es válida."},
}

// UseCases are the feedback use cases the handlers call (*app.Service).
type UseCases interface {
	SendFeedback(ctx context.Context, a app.Actor, in app.FeedbackInput) (*domain.Feedback, error)
}

// Handler implements feedbackapi.StrictServerInterface.
type Handler struct{ uc UseCases }

var _ feedbackapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the feedback operations on mux, with their error mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	errs := httpx.NewErrorMapper(log, errorRules...)
	feedbackapi.HandlerWithOptions(
		feedbackapi.NewStrictHandlerWithOptions(&Handler{uc: uc}, []feedbackapi.StrictMiddlewareFunc{auth.RequirePrincipal[feedbackapi.StrictHandlerFunc]}, feedbackapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		feedbackapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// SendFeedback stores the comment of the token's user.
func (h *Handler) SendFeedback(ctx context.Context, req feedbackapi.SendFeedbackRequestObject) (feedbackapi.SendFeedbackResponseObject, error) {
	p, _ := auth.PrincipalFrom(ctx)
	f, err := h.uc.SendFeedback(ctx, app.Actor{TenantID: p.TenantID, UserID: p.UserID}, app.FeedbackInput{
		Message: req.Body.Message, Screen: req.Body.Screen, AppVersion: req.Body.AppVersion,
	})
	if err != nil {
		return nil, err
	}
	return feedbackapi.SendFeedback201JSONResponse{Id: f.ID()}, nil
}
