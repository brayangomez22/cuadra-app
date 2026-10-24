package auth

import (
	"context"
	"net/http"
)

// StrictHandler is the shape of the StrictHandlerFunc of each package
// oapi-codegen generates (a distinct type in each one).
type StrictHandler interface {
	~func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error)
}

// RequirePrincipal is a strict middleware that answers 401 when no principal
// reached the handlers. The API's auth middleware already demands one; this
// keeps a wiring mistake from running a use case without a tenant. Mount it
// as []<pkg>.StrictMiddlewareFunc{auth.RequirePrincipal[<pkg>.StrictHandlerFunc]}.
func RequirePrincipal[F StrictHandler](next F, _ string) F {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
		if _, ok := PrincipalFrom(ctx); !ok {
			unauthorized(w)
			return nil, nil
		}
		return next(ctx, w, r, req)
	}
}

// CostPermission lets a role see costs and margins: whoever may edit the
// catalog. It is a permission of the identity module's matrix.
const CostPermission = "catalog:write"

// SeesCosts reports whether the caller of ctx may see costs and margins.
func SeesCosts(ctx context.Context, can Authorizer) bool {
	p, ok := PrincipalFrom(ctx)
	return ok && can(p.Role, CostPermission)
}
