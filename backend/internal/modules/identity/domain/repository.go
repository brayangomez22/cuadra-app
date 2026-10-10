package domain

import (
	"context"
	"time"

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
	// CreateWithOwner stores a new tenant and its first user in one
	// transaction: either both are stored or neither is (sign-up).
	CreateWithOwner(ctx context.Context, t *Tenant, owner *User) error
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

// LoginCandidate is a user registered with a given email, in some tenant.
type LoginCandidate struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// MaxLoginCandidates bounds how many tenants a login checks for one email,
// and with it the password hashes verified per attempt.
const MaxLoginCandidates = 10

// LoginDirectory finds the users registered with an email across tenants.
// It is the only cross-tenant read: it runs before the tenant is known and
// returns ids only, never user data.
type LoginDirectory interface {
	// FindByEmail returns at most MaxLoginCandidates candidates, ordered by
	// tenant id; none when the email is not registered.
	FindByEmail(ctx context.Context, email Email) ([]LoginCandidate, error)
}

// RefreshTokenRepository persists refresh tokens, scoped to their tenant.
type RefreshTokenRepository interface {
	// Create stores a new token.
	Create(ctx context.Context, t *RefreshToken) error
	// GetByHash returns ErrRefreshTokenNotFound when no token of tenantID
	// has hash.
	GetByHash(ctx context.Context, tenantID uuid.UUID, hash []byte) (*RefreshToken, error)
	// Rotate stores the revocation of old and the new token next in one
	// transaction. It returns ErrRefreshTokenReused, storing nothing, when old
	// was already revoked (a concurrent rotation won).
	Rotate(ctx context.Context, old, next *RefreshToken) error
	// RevokeFamily revokes every unrevoked token of the family and returns
	// how many it revoked. Idempotent.
	RevokeFamily(ctx context.Context, tenantID, familyID uuid.UUID, now time.Time) (int64, error)
}
