// Package httpadapter exposes the receivables use cases over HTTP: it
// implements the strict server generated from the "receivables" operations
// of api/openapi.yaml (receivablesapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/http/receivablesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// UseCases are every receivables use case the handlers call (*app.Service).
type UseCases interface {
	CreditUseCases
	PaymentUseCases
	ReportUseCases
}

// Handler implements receivablesapi.StrictServerInterface with a handler
// per resource.
type Handler struct {
	creditHandlers
	paymentHandlers
	reportHandlers
}

var _ receivablesapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the receivables operations on mux, with the module's
// error mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	h := &Handler{
		creditHandlers: creditHandlers{uc: uc}, paymentHandlers: paymentHandlers{uc: uc}, reportHandlers: reportHandlers{uc: uc},
	}
	errs := httpx.NewErrorMapper(log, errorRules...)
	receivablesapi.HandlerWithOptions(
		receivablesapi.NewStrictHandlerWithOptions(h, []receivablesapi.StrictMiddlewareFunc{auth.RequirePrincipal[receivablesapi.StrictHandlerFunc]}, receivablesapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		receivablesapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
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
