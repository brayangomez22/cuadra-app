package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RefreshTokenTTL is how long a refresh token lives. Each rotation issues a
// successor with a fresh TTL (sliding session).
const RefreshTokenTTL = 7 * 24 * time.Hour

// refreshSecretBytes is the entropy of a refresh token's secret.
const refreshSecretBytes = 32

// RefreshToken is a long-lived, single-use credential to obtain new access
// tokens. Only the SHA-256 of its secret is stored: the secret has 256 bits
// of entropy, so a slow hash (argon2) adds nothing.
//
// Tokens issued from the same login form a family. Each use rotates the
// token: it is revoked and a successor of the same family is issued. Using a
// revoked token again means someone else holds a copy, so the use case
// revokes the whole family.
type RefreshToken struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	userID    uuid.UUID
	familyID  uuid.UUID
	hash      []byte
	expiresAt time.Time
	createdAt time.Time
	revokedAt *time.Time
}

// NewRefreshToken starts a new family for userID and returns the token and
// its text form, "<tenant id>.<secret base64url>". The text is shown to the
// client once and never stored. The tenant travels in it so the token can be
// looked up within its tenant (Row-Level Security) before anyone is
// authenticated.
func NewRefreshToken(tenantID, userID uuid.UUID, now time.Time) (*RefreshToken, string, error) {
	familyID, err := uuid.NewV7()
	if err != nil {
		return nil, "", err
	}
	return issueRefreshToken(tenantID, userID, familyID, now)
}

func issueRefreshToken(tenantID, userID, familyID uuid.UUID, now time.Time) (*RefreshToken, string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, "", err
	}
	secret := make([]byte, refreshSecretBytes)
	_, _ = rand.Read(secret) // crypto/rand.Read never returns an error.

	now = now.UTC()
	t, err := newRefreshToken(RefreshTokenSnapshot{
		ID:        id,
		TenantID:  tenantID,
		UserID:    userID,
		FamilyID:  familyID,
		Hash:      hashSecret(secret),
		ExpiresAt: now.Add(RefreshTokenTTL),
		CreatedAt: now,
	})
	if err != nil {
		return nil, "", err
	}
	return t, tenantID.String() + "." + base64.RawURLEncoding.EncodeToString(secret), nil
}

// ParseRefreshToken splits a token's text form into its tenant and the hash
// to look it up by. It checks the format only, not that the token exists.
func ParseRefreshToken(raw string) (tenantID uuid.UUID, hash []byte, err error) {
	tenantPart, secretPart, ok := strings.Cut(raw, ".")
	if !ok {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}
	tenantID, err = uuid.Parse(tenantPart)
	if err != nil || tenantID == uuid.Nil || tenantPart != tenantID.String() {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}
	secret, err := base64.RawURLEncoding.Strict().DecodeString(secretPart)
	if err != nil || len(secret) != refreshSecretBytes {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}
	return tenantID, hashSecret(secret), nil
}

func hashSecret(secret []byte) []byte {
	sum := sha256.Sum256(secret)
	return sum[:]
}

// ID returns the token ID (UUID v7).
func (t *RefreshToken) ID() uuid.UUID { return t.id }

// TenantID returns the tenant of the token's user.
func (t *RefreshToken) TenantID() uuid.UUID { return t.tenantID }

// UserID returns the user the token signs in.
func (t *RefreshToken) UserID() uuid.UUID { return t.userID }

// FamilyID identifies the chain of rotations started by one login.
func (t *RefreshToken) FamilyID() uuid.UUID { return t.familyID }

// Hash returns a copy of the SHA-256 of the secret.
func (t *RefreshToken) Hash() []byte { return append([]byte(nil), t.hash...) }

// ExpiresAt returns when the token stops being valid, in UTC.
func (t *RefreshToken) ExpiresAt() time.Time { return t.expiresAt }

// CreatedAt returns the issue time in UTC.
func (t *RefreshToken) CreatedAt() time.Time { return t.createdAt }

// RevokedAt returns when the token was revoked (rotated, logged out or
// revoked with its family), or nil.
func (t *RefreshToken) RevokedAt() *time.Time {
	if t.revokedAt == nil {
		return nil
	}
	revokedAt := *t.revokedAt
	return &revokedAt
}

// IsRevoked reports whether the token was revoked.
func (t *RefreshToken) IsRevoked() bool { return t.revokedAt != nil }

// IsExpired reports whether the token is expired at now.
func (t *RefreshToken) IsExpired(now time.Time) bool { return !now.Before(t.expiresAt) }

// Rotate revokes the token and issues its successor in the same family, with
// a fresh TTL. A revoked token cannot be rotated (ErrRefreshTokenReused), nor
// an expired one (ErrRefreshTokenExpired); on error the token is unchanged.
func (t *RefreshToken) Rotate(now time.Time) (*RefreshToken, string, error) {
	switch {
	case t.IsRevoked():
		return nil, "", ErrRefreshTokenReused
	case t.IsExpired(now):
		return nil, "", ErrRefreshTokenExpired
	}
	next, raw, err := issueRefreshToken(t.tenantID, t.userID, t.familyID, now)
	if err != nil {
		return nil, "", err
	}
	revokedAt := now.UTC()
	t.revokedAt = &revokedAt
	return next, raw, nil
}

// RefreshTokenSnapshot holds a stored token's fields, for RehydrateRefreshToken.
type RefreshTokenSnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	FamilyID  uuid.UUID
	Hash      []byte
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// RehydrateRefreshToken rebuilds a stored token, checking the same
// invariants as NewRefreshToken.
func RehydrateRefreshToken(s RefreshTokenSnapshot) (*RefreshToken, error) {
	return newRefreshToken(s)
}

func newRefreshToken(s RefreshTokenSnapshot) (*RefreshToken, error) {
	if s.ID == uuid.Nil || s.TenantID == uuid.Nil || s.UserID == uuid.Nil || s.FamilyID == uuid.Nil ||
		len(s.Hash) != sha256.Size {
		return nil, ErrInvalidRefreshToken
	}
	t := &RefreshToken{
		id:        s.ID,
		tenantID:  s.TenantID,
		userID:    s.UserID,
		familyID:  s.FamilyID,
		hash:      append([]byte(nil), s.Hash...),
		expiresAt: s.ExpiresAt.UTC(),
		createdAt: s.CreatedAt.UTC(),
	}
	if s.RevokedAt != nil {
		revokedAt := s.RevokedAt.UTC()
		t.revokedAt = &revokedAt
	}
	return t, nil
}
