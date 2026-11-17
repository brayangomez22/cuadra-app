// Package httpadapter exposes the audit log over HTTP: it implements the
// strict server generated from the "audit" operations of api/openapi.yaml
// (auditapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/adapters/http/auditapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// Handler implements auditapi.StrictServerInterface.
type Handler struct {
	eventHandlers
}

var _ auditapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the audit operations on mux, with the module's error
// mapping.
func Register(mux *http.ServeMux, uc EventUseCases, log *slog.Logger) {
	h := &Handler{eventHandlers: eventHandlers{uc: uc}}
	errs := httpx.NewErrorMapper(log, errorRules...)
	auditapi.HandlerWithOptions(
		auditapi.NewStrictHandlerWithOptions(h, []auditapi.StrictMiddlewareFunc{auth.RequirePrincipal[auditapi.StrictHandlerFunc]}, auditapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		auditapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor returns who calls: tenant, user and role come only from the access
// token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID, Role: p.Role}
}
