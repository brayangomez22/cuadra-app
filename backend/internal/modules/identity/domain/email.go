package domain

import (
	"net/mail"
	"strings"
)

const (
	maxEmailLength      = 254 // RFC 5321 path limit
	maxEmailLocalLength = 64
)

// Email is a normalized (trimmed, lowercase) email address. The zero value is
// not a valid email; use ParseEmail.
type Email struct {
	value string
}

// ParseEmail normalizes s and validates it as a plain address: a local part,
// "@", and a domain with at least one dot. Display names ("Juan <j@x.co>"),
// quoted local parts and IP literals are rejected.
func ParseEmail(s string) (Email, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	if normalized == "" || len(normalized) > maxEmailLength {
		return Email{}, ErrInvalidEmail
	}

	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Name != "" || addr.Address != normalized {
		return Email{}, ErrInvalidEmail
	}

	local, domainPart, _ := strings.Cut(normalized, "@")
	if len(local) > maxEmailLocalLength || strings.HasPrefix(local, `"`) || !validEmailDomain(domainPart) {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: normalized}, nil
}

// validEmailDomain accepts dot-separated labels of [a-z0-9-] with a TLD.
func validEmailDomain(d string) bool {
	labels := strings.Split(d, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}

func (e Email) String() string { return e.value }

// IsZero reports whether e is the zero value (no address).
func (e Email) IsZero() bool { return e.value == "" }
