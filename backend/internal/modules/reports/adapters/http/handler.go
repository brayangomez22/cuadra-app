// Package httpadapter exposes the business reports over HTTP: it implements
// the strict server generated from the "reports" operations of
// api/openapi.yaml (reportsapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http/reportsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// UseCases are the reports' use cases (*app.Service).
type UseCases interface {
	SalesUseCases
	StockUseCases
	TeamUseCases
	ReceivablesAging(ctx context.Context, a app.Actor, sort string) (domain.AgingExport, error)
	Exported(ctx context.Context, report, format string)
}

// Handler implements reportsapi.StrictServerInterface.
type Handler struct {
	salesHandlers
	stockHandlers
	teamHandlers
	exportHandler
}

var _ reportsapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the report operations on mux, with the module's error
// mapping. can decides who sees costs and margins.
func Register(mux *http.ServeMux, uc UseCases, can auth.Authorizer, log *slog.Logger) {
	who := actors{can: can}
	h := &Handler{
		salesHandlers: salesHandlers{uc: uc, actors: who},
		stockHandlers: stockHandlers{uc: uc, actors: who},
		teamHandlers:  teamHandlers{uc: uc, actors: who},
		exportHandler: exportHandler{uc: uc, actors: who},
	}
	errs := httpx.NewErrorMapper(log, errorRules...)
	reportsapi.HandlerWithOptions(
		reportsapi.NewStrictHandlerWithOptions(h, []reportsapi.StrictMiddlewareFunc{auth.RequirePrincipal[reportsapi.StrictHandlerFunc]}, reportsapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		reportsapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actors builds who calls from the access token: tenant, user and role come
// only from it, and whether the role sees costs from the permission matrix.
type actors struct{ can auth.Authorizer }

func (w actors) actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID, Role: p.Role, SeesCosts: auth.SeesCosts(ctx, w.can)}
}

// page is the limit and offset of a request, with the contract's defaults.
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

// str is the value of an optional enum, or "".
func str[T ~string](v *T) string {
	if v == nil {
		return ""
	}
	return string(*v)
}
