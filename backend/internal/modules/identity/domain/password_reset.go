package domain

import (
	"crypto/sha256"
	"time"

	"github.com/google/uuid"
)

// PasswordReset lets a user choose a new password once, before it expires.
// An owner or admin issues it for someone who forgot theirs; only the
// SHA-256 of the link's secret is stored.
type PasswordReset struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	userID    uuid.UUID
	issuedBy  uuid.UUID
	hash      []byte
	expiresAt time.Time
	createdAt time.Time
	usedAt    *time.Time
}

// NewPasswordReset issues a reset of userID's password by issuedBy, valid
// for PasswordResetTTL, and returns it with its link token. The token is
// shown once and never stored.
func NewPasswordReset(tenantID, userID, issuedBy uuid.UUID, now time.Time) (*PasswordReset, string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, "", err
	}
	hash, text := newSecret(tenantID)
	r, err := newPasswordReset(PasswordResetSnapshot{
		ID: id, TenantID: tenantID, UserID: userID, IssuedBy: issuedBy,
		Hash: hash, ExpiresAt: now.Add(PasswordResetTTL), CreatedAt: now,
	})
	if err != nil {
		return nil, "", err
	}
	return r, text, nil
}

func newPasswordReset(s PasswordResetSnapshot) (*PasswordReset, error) {
	switch {
	case s.ID == uuid.Nil || s.UserID == uuid.Nil || s.IssuedBy == uuid.Nil:
		return nil, ErrInvalidUserID
	case s.TenantID == uuid.Nil:
		return nil, ErrInvalidTenantID
	case len(s.Hash) != sha256.Size:
		return nil, ErrInvalidLink
	}
	return &PasswordReset{
		id: s.ID, tenantID: s.TenantID, userID: s.UserID, issuedBy: s.IssuedBy,
		hash: append([]byte(nil), s.Hash...), expiresAt: s.ExpiresAt.UTC(), createdAt: s.CreatedAt.UTC(),
		usedAt: copyTime(s.UsedAt),
	}, nil
}

// ID returns the reset ID (UUID v7).
func (r *PasswordReset) ID() uuid.UUID { return r.id }

// TenantID returns the tenant of the user.
func (r *PasswordReset) TenantID() uuid.UUID { return r.tenantID }

// UserID returns the user whose password is reset.
func (r *PasswordReset) UserID() uuid.UUID { return r.userID }

// IssuedBy returns the owner or admin who issued the reset.
func (r *PasswordReset) IssuedBy() uuid.UUID { return r.issuedBy }

// Hash returns a copy of the SHA-256 of the link's secret.
func (r *PasswordReset) Hash() []byte { return append([]byte(nil), r.hash...) }

// ExpiresAt returns when the link stops working, in UTC.
func (r *PasswordReset) ExpiresAt() time.Time { return r.expiresAt }

// Use spends the reset: ErrLinkUsed if it was used and ErrLinkExpired if it
// expired. On error it is unchanged.
func (r *PasswordReset) Use(now time.Time) error {
	if err := usableUntil(r.expiresAt, r.usedAt != nil, now); err != nil {
		return err
	}
	r.usedAt = timePtr(now)
	return nil
}

// PasswordResetSnapshot holds a reset's fields, to store it and to rebuild
// it with RehydratePasswordReset.
type PasswordResetSnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	IssuedBy  uuid.UUID
	Hash      []byte
	ExpiresAt time.Time
	CreatedAt time.Time
	UsedAt    *time.Time
}

// Snapshot returns the reset's fields.
func (r *PasswordReset) Snapshot() PasswordResetSnapshot {
	return PasswordResetSnapshot{
		ID: r.id, TenantID: r.tenantID, UserID: r.userID, IssuedBy: r.issuedBy,
		Hash: r.Hash(), ExpiresAt: r.expiresAt, CreatedAt: r.createdAt, UsedAt: copyTime(r.usedAt),
	}
}

// RehydratePasswordReset rebuilds a stored reset, checking the same
// invariants as NewPasswordReset.
func RehydratePasswordReset(s PasswordResetSnapshot) (*PasswordReset, error) {
	return newPasswordReset(s)
}
