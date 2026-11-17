package flags

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Change sets one flag, globally or for a tenant.
type Change struct {
	Key     string
	Enabled bool
	// By is the operator who made the change, and At when.
	By uuid.UUID
	At time.Time
}

// Exception is the value a tenant has for a flag, apart from the global one.
type Exception struct {
	TenantID   uuid.UUID
	TenantName string
	Key        string
	Enabled    bool
}

// Store keeps the global value of each flag and the exceptions per tenant.
// A key with no row is off; the store may hold keys the code no longer
// declares, which the evaluator ignores.
type Store interface {
	// Globals returns the global value of each flag that has one.
	Globals(ctx context.Context) (map[string]bool, error)
	// Exceptions returns the tenant's exceptions, by key.
	Exceptions(ctx context.Context, tenantID uuid.UUID) (map[string]bool, error)
	// AllExceptions lists the exceptions of every tenant, for the operators.
	AllExceptions(ctx context.Context) ([]Exception, error)
	// SetGlobal stores the global value of a flag.
	SetGlobal(ctx context.Context, c Change) error
	// SetException stores a tenant's exception; ErrUnknownTenant if the
	// tenant does not exist.
	SetException(ctx context.Context, tenantID uuid.UUID, c Change) error
	// ClearException removes a tenant's exception, if it has one.
	ClearException(ctx context.Context, tenantID uuid.UUID, key string) error
}

// ErrUnknownTenant is the error for an exception of a tenant that does not exist.
var ErrUnknownTenant = errors.New("flags: unknown tenant")
