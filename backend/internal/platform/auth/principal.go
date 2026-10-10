// Package auth authenticates API requests with JWT access tokens and
// authorizes them by permission. Which operations are public and which
// permission each one needs is declared in the OpenAPI contract; Enforce
// applies it.
//
// The package knows nothing about the identity module: the role → permission
// matrix is injected as an Authorizer, and tokens are issued for the identity
// use cases through JWT.Issue.
package auth

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// Principal is the authenticated caller of a request. Its tenant is the only
// source of tenant_id for business operations.
type Principal struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Role     string
}

type principalKey struct{}

// WithPrincipal returns a copy of ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom returns the principal of an authenticated request.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// PrincipalAttrs returns tenant_id and user_id as log attributes. It is meant
// to be registered as a logger.ContextExtractor.
func PrincipalAttrs(ctx context.Context) []slog.Attr {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return nil
	}
	return []slog.Attr{
		slog.String("tenant_id", p.TenantID.String()),
		slog.String("user_id", p.UserID.String()),
	}
}
