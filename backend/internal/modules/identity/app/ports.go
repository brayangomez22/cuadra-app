package app

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccessTokenIssuer issues short-lived access tokens. platform/auth's JWT
// implements it; the use cases never see the token format.
type AccessTokenIssuer interface {
	Issue(tenantID, userID uuid.UUID, role string) (token string, expiresAt time.Time, err error)
}

// RateLimiter decides whether one more attempt is allowed for key and, when
// it is not, how long to wait.
type RateLimiter interface {
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

// TenantSetup knows the business templates and prepares a new tenant's
// initial data in the other modules from one of them (units, categories,
// attributes, variant axes and active modules). internal/server satisfies it;
// SignUp calls SetUpTenant inside the sign-up transaction.
type TenantSetup interface {
	// Templates lists the templates, in the order they are presented.
	Templates() []BusinessTemplate
	// SetUpTenant applies the template to the new tenant.
	SetUpTenant(ctx context.Context, tenantID uuid.UUID, template string) error
}

// BusinessTemplate is a template a business can sign up with.
type BusinessTemplate struct {
	ID          string
	Name        string
	Description string
	// DefaultTaxRate is the IVA rate the tenant proposes to new products.
	DefaultTaxRate decimal.Decimal
}

// ModulePermission is a permission an activable module declares, with the
// roles other than the owner that get it.
type ModulePermission struct {
	Module     string
	Permission string
	Roles      []string
}

// ModuleRegistry is the registry of activable modules, as identity sees it:
// the permissions each one declares. internal/server satisfies it with
// platform/modules, which identity does not import.
type ModuleRegistry interface {
	Permissions() []ModulePermission
}

// ActiveModules tells which modules a tenant has active. internal/server satisfies
// it with platform/modules.
type ActiveModules interface {
	Active(ctx context.Context, tenantID uuid.UUID) ([]string, error)
}
