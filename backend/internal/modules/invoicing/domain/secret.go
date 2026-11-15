package domain

import (
	"log/slog"
	"strings"
	"unicode"
)

const (
	maxSecretLength = 4096
	redacted        = "[REDACTED]"
)

// Secret is a credential, such as the token of the tenant's organization at
// the provider. It never shows: printing it, logging it or encoding it to
// JSON gives [REDACTED], and its value sits behind a pointer, so printing a
// struct that holds it shows an address. Only Reveal reads it, for the one
// adapter that sends it. The zero value is no secret.
type Secret struct{ p *string }

// NewSecret checks a credential: a non empty string of at most 4096
// characters without spaces or control characters (ErrInvalidProviderToken).
func NewSecret(s string) (Secret, error) {
	if s == "" || len(s) > maxSecretLength ||
		strings.ContainsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return Secret{}, ErrInvalidProviderToken
	}
	return Secret{p: &s}, nil
}

// Reveal returns the credential, or "" for no secret.
func (s Secret) Reveal() string {
	if s.p == nil {
		return ""
	}
	return *s.p
}

// IsZero is whether there is no secret.
func (s Secret) IsZero() bool { return s.p == nil }

func (s Secret) String() string { return redacted }

// GoString hides the secret from %#v.
func (s Secret) GoString() string { return redacted }

// LogValue hides the secret from slog.
func (s Secret) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalText hides the secret from encoding/json and the text encoders.
func (s Secret) MarshalText() ([]byte, error) { return []byte(redacted), nil }
