// Package httpadapter exposes the operator screen of the feature flags over
// HTTP: it implements the strict server generated from the "flags"
// operations of api/openapi.yaml (flagsapi) and only translates HTTP ↔ use
// case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags/adapters/http/flagsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the flag errors to HTTP. A user who is not an operator
// gets the 404 of a route that does not exist: the screen is not announced.
var errorRules = []httpx.ErrorRule{
	{Err: flags.ErrNotOperator, Status: http.StatusNotFound, Code: "not_found", Message: "El recurso no existe."},
	{Err: flags.ErrUnknownFlag, Status: http.StatusNotFound, Code: "unknown_flag", Message: "La flag no existe."},
	{Err: flags.ErrUnknownTenant, Status: http.StatusNotFound, Code: "unknown_tenant", Message: "La empresa no existe."},
}

// UseCases are the flag use cases the handlers call (*flags.Service).
type UseCases interface {
	List(ctx context.Context, userID uuid.UUID) ([]flags.Status, error)
	SetGlobal(ctx context.Context, userID uuid.UUID, key string, enabled bool) (flags.Status, error)
	SetException(ctx context.Context, userID, tenantID uuid.UUID, key string, enabled bool) error
	ClearException(ctx context.Context, userID, tenantID uuid.UUID, key string) error
}

// Handler implements flagsapi.StrictServerInterface.
type Handler struct{ uc UseCases }

var _ flagsapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the flag operations on mux, with their error mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	errs := httpx.NewErrorMapper(log, errorRules...)
	flagsapi.HandlerWithOptions(
		flagsapi.NewStrictHandlerWithOptions(&Handler{uc: uc}, []flagsapi.StrictMiddlewareFunc{auth.RequirePrincipal[flagsapi.StrictHandlerFunc]}, flagsapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		flagsapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// ListFlags lists every flag with its global value and exceptions.
func (h *Handler) ListFlags(ctx context.Context, _ flagsapi.ListFlagsRequestObject) (flagsapi.ListFlagsResponseObject, error) {
	p, _ := auth.PrincipalFrom(ctx)
	list, err := h.uc.List(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	items := make([]flagsapi.FeatureFlag, 0, len(list))
	for _, st := range list {
		exceptions := make([]flagsapi.FeatureFlagException, 0, len(st.Exceptions))
		for _, ex := range st.Exceptions {
			exceptions = append(exceptions, flagsapi.FeatureFlagException{
				TenantId: ex.TenantID, TenantName: ex.TenantName, Enabled: ex.Enabled,
			})
		}
		items = append(items, flagsapi.FeatureFlag{
			Key: st.Key, Description: st.Description, Enabled: st.Global, Exceptions: exceptions,
		})
	}
	return flagsapi.ListFlags200JSONResponse{Items: items}, nil
}

// SetFlagGlobal sets a flag's global value.
func (h *Handler) SetFlagGlobal(ctx context.Context, req flagsapi.SetFlagGlobalRequestObject) (flagsapi.SetFlagGlobalResponseObject, error) {
	p, _ := auth.PrincipalFrom(ctx)
	st, err := h.uc.SetGlobal(ctx, p.UserID, req.FlagKey, req.Body.Enabled)
	if err != nil {
		return nil, err
	}
	return flagsapi.SetFlagGlobal200JSONResponse{Key: st.Key, Description: st.Description, Enabled: st.Global}, nil
}

// SetFlagException sets a flag's value for one tenant, which the operator
// names in the path.
func (h *Handler) SetFlagException(ctx context.Context, req flagsapi.SetFlagExceptionRequestObject) (flagsapi.SetFlagExceptionResponseObject, error) {
	p, _ := auth.PrincipalFrom(ctx)
	if err := h.uc.SetException(ctx, p.UserID, req.TenantId, req.FlagKey, req.Body.Enabled); err != nil {
		return nil, err
	}
	return flagsapi.SetFlagException204Response{}, nil
}

// ClearFlagException removes a tenant's exception.
func (h *Handler) ClearFlagException(ctx context.Context, req flagsapi.ClearFlagExceptionRequestObject) (flagsapi.ClearFlagExceptionResponseObject, error) {
	p, _ := auth.PrincipalFrom(ctx)
	if err := h.uc.ClearException(ctx, p.UserID, req.TenantId, req.FlagKey); err != nil {
		return nil, err
	}
	return flagsapi.ClearFlagException204Response{}, nil
}
