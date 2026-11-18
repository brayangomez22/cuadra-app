package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

// secretBytes is the entropy of the secret of a refresh token or a link.
const secretBytes = 32

// Lifetimes of the links a user receives to join a tenant or to choose a
// new password.
const (
	InvitationTTL    = 7 * 24 * time.Hour
	PasswordResetTTL = 24 * time.Hour
)

// newSecret returns a random secret's SHA-256, which is what is stored,
// and its text form, "<tenant id>.<secret base64url>", which is shown to
// the client once. The secret has 256 bits of entropy, so a slow hash
// (argon2) adds nothing. The tenant travels in the text so the secret can
// be looked up within its tenant (Row-Level Security) before anyone is
// authenticated.
func newSecret(tenantID uuid.UUID) (hash []byte, text string) {
	secret := make([]byte, secretBytes)
	_, _ = rand.Read(secret) // crypto/rand.Read never returns an error.
	return hashSecret(secret), tenantID.String() + "." + base64.RawURLEncoding.EncodeToString(secret)
}

// parseSecret splits a secret's text form into its tenant and its hash; a
// malformed text is invalid.
func parseSecret(raw string, invalid error) (tenantID uuid.UUID, hash []byte, err error) {
	tenantPart, secretPart, ok := strings.Cut(raw, ".")
	if !ok {
		return uuid.Nil, nil, invalid
	}
	tenantID, err = uuid.Parse(tenantPart)
	if err != nil || tenantID == uuid.Nil || tenantPart != tenantID.String() {
		return uuid.Nil, nil, invalid
	}
	secret, err := base64.RawURLEncoding.Strict().DecodeString(secretPart)
	if err != nil || len(secret) != secretBytes {
		return uuid.Nil, nil, invalid
	}
	return tenantID, hashSecret(secret), nil
}

func hashSecret(secret []byte) []byte {
	sum := sha256.Sum256(secret)
	return sum[:]
}

// ParseLinkToken splits the token of an invitation or password reset link
// into its tenant and the hash to look it up by. It checks the format
// only (ErrInvalidLink), not that the link exists.
func ParseLinkToken(raw string) (tenantID uuid.UUID, hash []byte, err error) {
	return parseSecret(raw, ErrInvalidLink)
}

// usableUntil checks a link that expires at expiresAt and was used or
// revoked when done is true.
func usableUntil(expiresAt time.Time, done bool, now time.Time) error {
	switch {
	case done:
		return ErrLinkUsed
	case !now.Before(expiresAt):
		return ErrLinkExpired
	}
	return nil
}

// timePtr returns a pointer to a UTC copy of t.
func timePtr(t time.Time) *time.Time {
	u := t.UTC()
	return &u
}

// copyTime returns a UTC copy of *t, or nil.
func copyTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	return timePtr(*t)
}
