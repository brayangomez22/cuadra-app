// Package httpadapter exposes the customers use cases over HTTP: it
// implements the strict server generated from the "customers" operations of
// api/openapi.yaml (customersapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/http/customersapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// UseCases are every customers use case the handlers call (*app.Service).
type UseCases interface {
	CustomerUseCases
	SegmentUseCases
}

// Handler implements customersapi.StrictServerInterface with a handler per
// resource.
type Handler struct {
	customerHandlers
	segmentHandlers
}

var _ customersapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the customers operations on mux, with the module's error
// mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	h := &Handler{customerHandlers: customerHandlers{uc: uc}, segmentHandlers: segmentHandlers{uc: uc}}
	errs := httpx.NewErrorMapper(log, errorRules...)
	customersapi.HandlerWithOptions(
		customersapi.NewStrictHandlerWithOptions(h, []customersapi.StrictMiddlewareFunc{auth.RequirePrincipal[customersapi.StrictHandlerFunc]}, customersapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		customersapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor returns who calls: tenant and user come only from the access token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}
}
