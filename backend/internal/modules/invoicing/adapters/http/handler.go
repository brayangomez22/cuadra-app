// Package httpadapter exposes the invoicing use cases over HTTP: it
// implements the strict server generated from the "invoicing" operations of
// api/openapi.yaml (invoicingapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/http/invoicingapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// UseCases are every invoicing use case the handlers call (*app.Service).
type UseCases interface {
	SettingsUseCases
	ResolutionUseCases
	DocumentUseCases
}

// Handler implements invoicingapi.StrictServerInterface with a handler per
// resource.
type Handler struct {
	settingsHandlers
	resolutionHandlers
	documentHandlers
}

var _ invoicingapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the invoicing operations on mux, with the module's error
// mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	h := &Handler{
		settingsHandlers: settingsHandlers{uc: uc}, resolutionHandlers: resolutionHandlers{uc: uc}, documentHandlers: documentHandlers{uc: uc},
	}
	errs := httpx.NewErrorMapper(log, errorRules...)
	invoicingapi.HandlerWithOptions(
		invoicingapi.NewStrictHandlerWithOptions(h, []invoicingapi.StrictMiddlewareFunc{auth.RequirePrincipal[invoicingapi.StrictHandlerFunc]}, invoicingapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		invoicingapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor returns who calls: tenant and user come only from the access token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}
}

// page returns the limit and offset of a request, with the contract's
// defaults.
func page(limit, offset *int) (int, int) {
	l, o := defaultPageSize, 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}
	return l, o
}

// optional is s, or nil when it is empty.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
