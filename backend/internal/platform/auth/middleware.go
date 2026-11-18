package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// User-facing messages of the auth errors.
const (
	unauthorizedMessage = "Su sesión no es válida o expiró. Inicie sesión de nuevo."
	forbiddenMessage    = "No tiene permiso para realizar esta acción."
)

// Verifier checks an access token and returns its principal.
type Verifier interface {
	Verify(token string) (Principal, error)
}

// ErrSessionEnded means the token is valid but its user can no longer act:
// they were deactivated or deleted.
var ErrSessionEnded = errors.New("auth: session ended")

// Sessions checks, on every request, that the user of a valid token is
// still active and returns the principal with their current role: a
// deactivation or a role change applies at once instead of when the token
// expires. The identity module implements it; the wiring injects it.
type Sessions interface {
	// Current returns p with its current role, or ErrSessionEnded.
	Current(ctx context.Context, p Principal) (Principal, error)
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
func Enforce(policy Policy, v Verifier, s Sessions, can Authorizer, log *slog.Logger) func(http.Handler) http.Handler {
	requireAuth := RequireAuth(v, s, log)
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

// RequireAuth rejects requests without a valid Bearer access token, or whose
// user is no longer active (401), and puts the principal, with the user's
// current role, in the context of the rest. The tenant, user and role
// are also set on the request's span, and the tenant and user on the request's
// log line (httpx.Logging). The token itself is never logged.
func RequireAuth(v Verifier, s Sessions, log *slog.Logger) func(http.Handler) http.Handler {
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
			p, ok = current(w, r, s, p, log)
			if !ok {
				return
			}
			trace.SpanFromContext(r.Context()).SetAttributes(
				attribute.String("tenant.id", p.TenantID.String()),
				attribute.String("user.id", p.UserID.String()),
				attribute.String("user.role", p.Role),
			)
			ctx := WithPrincipal(r.Context(), p)
			httpx.AddRequestLogAttrs(ctx, PrincipalAttrs(ctx)...)
			next.ServeHTTP(w, r.WithContext(ctx))
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

// current asks s for p's current role; on failure it answers the request
// (401 if the session ended, 500 otherwise) and returns false.
func current(w http.ResponseWriter, r *http.Request, s Sessions, p Principal, log *slog.Logger) (Principal, bool) {
	cur, err := s.Current(r.Context(), p)
	switch {
	case errors.Is(err, ErrSessionEnded):
		log.InfoContext(r.Context(), "access token of an inactive user rejected",
			slog.String("tenant_id", p.TenantID.String()), slog.String("user_id", p.UserID.String()))
		unauthorized(w)
		return Principal{}, false
	case err != nil:
		log.ErrorContext(r.Context(), "session check failed", slog.Any("error", err))
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", httpx.InternalErrorMessage)
		return Principal{}, false
	}
	return cur, true
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
