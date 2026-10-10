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
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newUser(UserSnapshot{
		ID:           id,
		TenantID:     tenantID,
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		Role:         role,
		Active:       true,
		CreatedAt:    now,
	})
}

// newUser checks the invariants shared by NewUser and RehydrateUser.
func newUser(s UserSnapshot) (*User, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if s.Email.IsZero() {
		return nil, ErrInvalidEmail
	}
	name, err := normalizeName(s.Name, ErrInvalidUserName)
	if err != nil {
		return nil, err
	}
	if s.PasswordHash == "" {
		return nil, ErrEmptyPasswordHash
	}
	if !s.Role.valid() {
		return nil, ErrInvalidRole
	}
	return &User{
		id:           s.ID,
		tenantID:     s.TenantID,
		email:        s.Email,
		name:         name,
		passwordHash: s.PasswordHash,
		role:         s.Role,
		active:       s.Active,
		createdAt:    s.CreatedAt.UTC(),
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

// UserSnapshot holds a stored user's fields, for RehydrateUser.
type UserSnapshot struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Email        Email
	Name         string
	PasswordHash string
	Role         Role
	Active       bool
	CreatedAt    time.Time
}

// RehydrateUser rebuilds a stored user, checking the same invariants as
// NewUser so corrupt data surfaces as an error instead of an invalid entity.
func RehydrateUser(s UserSnapshot) (*User, error) { return newUser(s) }
