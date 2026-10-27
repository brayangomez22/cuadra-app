// Package httpadapter exposes the imports use cases over HTTP: it implements
// the strict server generated from the "imports" operations of
// api/openapi.yaml (importsapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/http/importsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// UseCases are the imports use cases the handlers call (*app.Service).
type UseCases interface {
	UploadImport(ctx context.Context, a app.Actor, in app.UploadInput) (app.Preview, error)
	ConfirmImport(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Import, error)
	GetImport(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Import, error)
	ListImportErrors(ctx context.Context, a app.Actor, id uuid.UUID, limit, offset int) (app.ErrorPage, error)
	ImportTemplate(ctx context.Context, a app.Actor, format domain.Format) ([]byte, error)
}

// Handler implements importsapi.StrictServerInterface.
type Handler struct{ uc UseCases }

var _ importsapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the imports operations on mux, with the module's error
// mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	errs := httpx.NewErrorMapper(log, errorRules...)
	middlewares := []importsapi.StrictMiddlewareFunc{auth.RequirePrincipal[importsapi.StrictHandlerFunc]}
	importsapi.HandlerWithOptions(
		importsapi.NewStrictHandlerWithOptions(&Handler{uc: uc}, middlewares, importsapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		importsapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor returns who calls: tenant and user come only from the access token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}
}

// GetImport returns an import of the tenant.
func (h *Handler) GetImport(ctx context.Context, req importsapi.GetImportRequestObject) (importsapi.GetImportResponseObject, error) {
	imp, err := h.uc.GetImport(ctx, actor(ctx), req.ImportId)
	if err != nil {
		return nil, err
	}
	return importsapi.GetImport200JSONResponse(importBody(imp)), nil
}

// ConfirmImport queues a validated import.
func (h *Handler) ConfirmImport(ctx context.Context, req importsapi.ConfirmImportRequestObject) (importsapi.ConfirmImportResponseObject, error) {
	imp, err := h.uc.ConfirmImport(ctx, actor(ctx), req.ImportId)
	if err != nil {
		return nil, err
	}
	return importsapi.ConfirmImport200JSONResponse(importBody(imp)), nil
}

// ListImportErrors returns one page of an import's row errors.
func (h *Handler) ListImportErrors(ctx context.Context, req importsapi.ListImportErrorsRequestObject) (importsapi.ListImportErrorsResponseObject, error) {
	limit, offset := 0, 0
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	if req.Params.Offset != nil {
		offset = *req.Params.Offset
	}
	page, err := h.uc.ListImportErrors(ctx, actor(ctx), req.ImportId, limit, offset)
	if err != nil {
		return nil, err
	}
	return importsapi.ListImportErrors200JSONResponse{Items: rowErrorsBody(page.Items), Total: page.Total}, nil
}
