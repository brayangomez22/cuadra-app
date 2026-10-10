package auth

import (
	"log/slog"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// User-facing messages of the auth errors.
const (
	unauthorizedMessage = "Tu sesión no es válida o expiró. Inicia sesión de nuevo."
	forbiddenMessage    = "No tienes permiso para realizar esta acción."
)

// Verifier checks an access token and returns its principal.
type Verifier interface {
	Verify(token string) (Principal, error)
}

// Authorizer reports whether role grants permission. The identity module
// owns the matrix; the wiring injects it.
type Authorizer func(role, permission string) bool

// Requirement is what an operation demands of the caller.
type Requirement struct {
	// Public operations need no token.
	Public bool
	// Permission, when set, must be granted by the caller's role.
	Permission string
}

// Policy returns the requirement of the operation a request is routed to;
// ok is false when it does not know the operation.
type Policy func(r *http.Request) (req Requirement, ok bool)

// Enforce applies policy to every request. It is secure by default:
// operations the policy does not know, and operations not marked public,
// require a valid token.
func Enforce(policy Policy, v Verifier, can Authorizer, log *slog.Logger) func(http.Handler) http.Handler {
	requireAuth := RequireAuth(v, log)
	return func(next http.Handler) http.Handler {
		authenticated := requireAuth(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			req, ok := policy(r)
			switch {
			case ok && req.Public:
				next.ServeHTTP(w, r)
			case req.Permission == "":
				authenticated.ServeHTTP(w, r)
			default:
				requireAuth(RequirePermission(can, req.Permission, log)(next)).ServeHTTP(w, r)
			}
		})
	}
}

// RequireAuth rejects requests without a valid Bearer access token (401) and
// puts the principal in the context of the rest. The tenant, user and role
// are also set on the request's span. The token itself is never logged.
func RequireAuth(v Verifier, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				unauthorized(w)
				return
			}
			p, err := v.Verify(token)
			if err != nil {
				// The cause comes from the JWT parser; it never includes the token.
				log.InfoContext(r.Context(), "access token rejected", slog.Any("error", err))
				unauthorized(w)
				return
			}
			trace.SpanFromContext(r.Context()).SetAttributes(
				attribute.String("tenant.id", p.TenantID.String()),
				attribute.String("user.id", p.UserID.String()),
				attribute.String("user.role", p.Role),
			)
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

// RequirePermission rejects requests whose principal's role does not grant
// permission (403). It must run after RequireAuth; without a principal it
// answers 401.
func RequirePermission(can Authorizer, permission string, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok {
				unauthorized(w)
				return
			}
			if !can(p.Role, permission) {
				log.InfoContext(r.Context(), "permission denied",
					slog.String("permission", permission), slog.String("role", p.Role))
				httpx.WriteError(w, http.StatusForbidden, "forbidden", forbiddenMessage)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", unauthorizedMessage)
}
