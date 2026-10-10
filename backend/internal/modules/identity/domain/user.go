package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is a person who signs in to a tenant. It stores only the password
// hash, never the password.
type User struct {
	id           uuid.UUID
	tenantID     uuid.UUID
	email        Email
	name         string
	passwordHash string
	role         Role
	active       bool
	createdAt    time.Time
}

// NewUser creates an active user. passwordHash must come from a
// PasswordHasher, after ValidatePassword.
func NewUser(tenantID uuid.UUID, email Email, name, passwordHash string, role Role, now time.Time) (*User, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if email.IsZero() {
		return nil, ErrInvalidEmail
	}
	name, err := normalizeName(name, ErrInvalidUserName)
	if err != nil {
		return nil, err
	}
	if passwordHash == "" {
		return nil, ErrEmptyPasswordHash
	}
	if !role.valid() {
		return nil, ErrInvalidRole
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return &User{
		id:           id,
		tenantID:     tenantID,
		email:        email,
		name:         name,
		passwordHash: passwordHash,
		role:         role,
		active:       true,
		createdAt:    now.UTC(),
	}, nil
}

// ID returns the user ID (UUID v7).
func (u *User) ID() uuid.UUID { return u.id }

// TenantID returns the tenant the user belongs to.
func (u *User) TenantID() uuid.UUID { return u.tenantID }

// Email returns the normalized email.
func (u *User) Email() Email { return u.email }

// Name returns the display name.
func (u *User) Name() string { return u.name }

// PasswordHash returns the encoded password hash.
func (u *User) PasswordHash() string { return u.passwordHash }

// Role returns the role within the tenant.
func (u *User) Role() Role { return u.role }

// IsActive reports whether the user may sign in.
func (u *User) IsActive() bool { return u.active }

// CreatedAt returns the creation time in UTC.
func (u *User) CreatedAt() time.Time { return u.createdAt }

// Can reports whether the user may perform p. An inactive user can do nothing.
func (u *User) Can(p Permission) bool { return u.active && u.role.Can(p) }

// Deactivate blocks the user without deleting it. Idempotent.
func (u *User) Deactivate() { u.active = false }

// Activate reactivates the user. Idempotent.
func (u *User) Activate() { u.active = true }

// ChangeRole assigns a new role; the user keeps its role if r is invalid.
func (u *User) ChangeRole(r Role) error {
	if !r.valid() {
		return ErrInvalidRole
	}
	u.role = r
	return nil
}
