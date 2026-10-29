package httpadapter

import (
	"context"
	"errors"
	"math"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http/identityapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
)

// DemoUseCases are the use cases of the public demo.
type DemoUseCases interface {
	StartDemo(ctx context.Context, in app.DemoLoginInput) (app.Session, error)
	DemoOptions(ctx context.Context) ([]app.BusinessTemplate, error)
}

// ListDemos lists the business types that have a demo.
func (h *Handler) ListDemos(ctx context.Context, _ identityapi.ListDemosRequestObject) (identityapi.ListDemosResponseObject, error) {
	options, err := h.uc.DemoOptions(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]identityapi.DemoOption, 0, len(options))
	for _, t := range options {
		items = append(items, identityapi.DemoOption{BusinessTemplate: t.ID, Name: t.Name})
	}
	return identityapi.ListDemos200JSONResponse{Items: items}, nil
}

// StartDemo starts a session as the chosen role of a demo tenant.
func (h *Handler) StartDemo(ctx context.Context, req identityapi.StartDemoRequestObject) (identityapi.StartDemoResponseObject, error) {
	session, err := h.uc.StartDemo(ctx, app.DemoLoginInput{
		BusinessTemplate: req.Body.BusinessTemplate, Role: string(req.Body.Role), ClientIP: clientIP(ctx),
	})
	var limited *app.RateLimitError
	switch {
	case errors.As(err, &limited):
		retryAfter := int(math.Ceil(limited.RetryAfter.Seconds()))
		return identityapi.StartDemo429JSONResponse{
			Body:    errorBody("too_many_requests", tooManyRequestsMessage),
			Headers: identityapi.StartDemo429ResponseHeaders{RetryAfter: &retryAfter},
		}, nil
	case err != nil:
		return nil, err
	}
	body, cookie := h.authSession(session)
	return identityapi.StartDemo200JSONResponse{Body: body, Headers: identityapi.StartDemo200ResponseHeaders{SetCookie: &cookie}}, nil
}
