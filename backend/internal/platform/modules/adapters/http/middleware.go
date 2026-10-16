package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// ActiveChecker tells whether a tenant has a module active (*modules.Service).
type ActiveChecker interface {
	IsActive(ctx context.Context, tenantID uuid.UUID, id string) (bool, error)
}

// RequireModule answers 404 when the tenant of the request's principal does
// not have the module id active: to the client, an operation of an inactive
// module is a route that does not exist. It must run after RequireAuth;
// without a principal it answers 401.
func RequireModule(modules ActiveChecker, id string, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := auth.PrincipalFrom(r.Context())
			if !ok {
				w.Header().Set("WWW-Authenticate", "Bearer")
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", unauthorizedMessage)
				return
			}
			active, err := modules.IsActive(r.Context(), p.TenantID, id)
			if err != nil {
				log.ErrorContext(r.Context(), "check active module", slog.String("module_id", id), slog.Any("error", err))
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", httpx.InternalErrorMessage)
				return
			}
			if !active {
				httpx.WriteError(w, http.StatusNotFound, "not_found", "El recurso no existe.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
