// Package httpadapter exposes the module service over HTTP: it implements
// the strict server generated from the "modules" operations of
// api/openapi.yaml (modulesapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules/adapters/http/modulesapi"
)

// unauthorizedMessage answers a request that reaches the handlers without a
// principal, which RequireAuth should have stopped.
const unauthorizedMessage = "Su sesión no es válida o expiró. Inicie sesión de nuevo."

// errorRules maps the module errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: modules.ErrUnknownModule, Status: http.StatusNotFound, Code: "unknown_module", Message: "El módulo no existe."},
	{Err: modules.ErrDependencyInactive, Status: http.StatusConflict, Code: "module_dependency_inactive",
		Message: "Primero active el módulo del que depende este.",
		Describe: describeDependency(func(name string) string {
			return "Primero active el módulo «" + name + "»: este depende de él."
		})},
	{Err: modules.ErrRequiredBy, Status: http.StatusConflict, Code: "module_required_by",
		Message: "Otro módulo activo depende de este. Desactívelo primero.",
		Describe: describeDependency(func(name string) string {
			return "El módulo «" + name + "» depende de este. Desactívelo primero."
		})},
}

func describeDependency(format func(name string) string) func(error) (string, map[string]string, bool) {
	return func(err error) (string, map[string]string, bool) {
		var depErr *modules.DependencyError
		if !errors.As(err, &depErr) {
			return "", nil, false
		}
		return format(depErr.Name), map[string]string{"module": depErr.Module}, true
	}
}

// UseCases are the module use cases the handlers call (*modules.Service).
type UseCases interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]modules.Status, error)
	SetActive(ctx context.Context, tenantID, userID uuid.UUID, id string, active bool) (modules.Status, error)
}

// Handler implements modulesapi.StrictServerInterface.
type Handler struct{ uc UseCases }

var _ modulesapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the module operations on mux, with their error mapping.
func Register(mux *http.ServeMux, uc UseCases, log *slog.Logger) {
	errs := httpx.NewErrorMapper(log, errorRules...)
	modulesapi.HandlerWithOptions(
		modulesapi.NewStrictHandlerWithOptions(&Handler{uc: uc}, []modulesapi.StrictMiddlewareFunc{auth.RequirePrincipal[modulesapi.StrictHandlerFunc]}, modulesapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		modulesapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// ListModules lists every module with whether the tenant has it active.
func (h *Handler) ListModules(ctx context.Context, _ modulesapi.ListModulesRequestObject) (modulesapi.ListModulesResponseObject, error) {
	p, _ := auth.PrincipalFrom(ctx)
	list, err := h.uc.List(ctx, p.TenantID)
	if err != nil {
		return nil, err
	}
	items := make([]modulesapi.Module, 0, len(list))
	for _, st := range list {
		items = append(items, module(st))
	}
	return modulesapi.ListModules200JSONResponse{Items: items}, nil
}

// SetModuleActive turns a module on or off for the tenant of the token.
func (h *Handler) SetModuleActive(ctx context.Context, req modulesapi.SetModuleActiveRequestObject) (modulesapi.SetModuleActiveResponseObject, error) {
	p, _ := auth.PrincipalFrom(ctx)
	st, err := h.uc.SetActive(ctx, p.TenantID, p.UserID, req.ModuleId, req.Body.Active)
	if err != nil {
		return nil, err
	}
	return modulesapi.SetModuleActive200JSONResponse(module(st)), nil
}

func module(st modules.Status) modulesapi.Module {
	dependsOn := st.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}
	return modulesapi.Module{Id: st.ID, Name: st.Name, DependsOn: dependsOn, Active: st.Active}
}
