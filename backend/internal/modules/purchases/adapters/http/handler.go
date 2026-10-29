// Package httpadapter exposes the purchases use cases over HTTP: it
// implements the strict server generated from the "purchases" operations of
// api/openapi.yaml (purchasesapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/http/purchasesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// UseCases are every purchases use case the handlers call (*app.Service).
type UseCases interface {
	SupplierUseCases
	PurchaseUseCases
}

// Handler implements purchasesapi.StrictServerInterface with a handler per
// resource.
type Handler struct {
	supplierHandlers
	purchaseHandlers
}

var _ purchasesapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the purchases operations on mux, with the module's error
// mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	h := &Handler{supplierHandlers: supplierHandlers{uc: uc}, purchaseHandlers: purchaseHandlers{uc: uc}}
	errs := httpx.NewErrorMapper(log, errorRules...)
	purchasesapi.HandlerWithOptions(
		purchasesapi.NewStrictHandlerWithOptions(h, []purchasesapi.StrictMiddlewareFunc{auth.RequirePrincipal[purchasesapi.StrictHandlerFunc]}, purchasesapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		purchasesapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor returns who calls: tenant and user come only from the access token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}
}

func date(d domain.Date) openapi_types.Date { return openapi_types.Date{Time: d.Time()} }

func dateOf(d *openapi_types.Date) *domain.Date {
	if d == nil {
		return nil
	}
	return new(domain.DateOf(d.Time))
}

// optional returns nil for "", so empty fields are left out of a body.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func text(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
