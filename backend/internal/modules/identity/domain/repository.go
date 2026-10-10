package domain

import (
	"context"

	"github.com/google/uuid"
)

// TenantRepository persists tenants. Implementations scope every operation to
// the tenant itself (Row-Level Security).
type TenantRepository interface {
	// Create stores a new tenant.
	Create(ctx context.Context, t *Tenant) error
	// GetByID returns ErrTenantNotFound when the tenant does not exist.
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	// Update stores the name and status; ErrTenantNotFound if it does not exist.
	Update(ctx context.Context, t *Tenant) error
}

// UserRepository persists users. Every read is scoped to tenantID: a user of
// another tenant is reported as ErrUserNotFound.
type UserRepository interface {
	// Create stores a new user; ErrEmailTaken if the email is already
	// registered in the tenant.
	Create(ctx context.Context, u *User) error
	// GetByID returns ErrUserNotFound when the user does not exist in tenantID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*User, error)
	// GetByEmail returns ErrUserNotFound when no user of tenantID has email.
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email Email) (*User, error)
	// Update stores the name, password hash, role and active flag;
	// ErrUserNotFound if the user does not exist in its tenant.
	Update(ctx context.Context, u *User) error
}
