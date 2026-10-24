// Package httpadapter exposes the catalog use cases over HTTP: it implements
// the strict server generated from the "catalog" operations of
// api/openapi.yaml (catalogapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// UseCases are the catalog use cases the handlers call (*app.Service), one
// small interface per resource.
type UseCases interface {
	ProductUseCases
	VariantUseCases
	CategoryUseCases
	UnitUseCases
	AttributeUseCases
	VariantAxisUseCases
}

// Handler implements catalogapi.StrictServerInterface with one handler per
// resource.
type Handler struct {
	productHandlers
	variantHandlers
	categoryHandlers
	unitHandlers
	attributeHandlers
	variantAxisHandlers
}

var _ catalogapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the catalog operations on mux, with the module's error
// mapping. can decides who sees a product's cost and margin.
func Register(mux *http.ServeMux, uc UseCases, can auth.Authorizer, log *slog.Logger) {
	costs := costView{can: can}
	h := &Handler{
		productHandlers:     productHandlers{uc: uc, costView: costs},
		variantHandlers:     variantHandlers{uc: uc, costView: costs},
		categoryHandlers:    categoryHandlers{uc: uc},
		unitHandlers:        unitHandlers{uc: uc},
		attributeHandlers:   attributeHandlers{uc: uc},
		variantAxisHandlers: variantAxisHandlers{uc: uc},
	}
	errs := httpx.NewErrorMapper(log, errorRules...)
	middlewares := []catalogapi.StrictMiddlewareFunc{auth.RequirePrincipal[catalogapi.StrictHandlerFunc]}
	catalogapi.HandlerWithOptions(
		catalogapi.NewStrictHandlerWithOptions(h, middlewares, catalogapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		catalogapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor returns who calls: tenant and user come only from the access token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}
}

// costView decides who sees costs and margins.
type costView struct{ can auth.Authorizer }

// seesCost reports whether the caller may see costs and margins.
func (c costView) seesCost(ctx context.Context) bool { return auth.SeesCosts(ctx, c.can) }
