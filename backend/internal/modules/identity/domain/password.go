package domain

import "unicode/utf8"

// Password length limits, in characters (NIST SP 800-63B: length over
// composition rules). The maximum bounds the hashing cost per request.
const (
	MinPasswordLength = 8
	MaxPasswordLength = 72
)

// PasswordHasher hashes and verifies passwords. The implementation lives in
// an adapter (argon2id); the domain only stores the encoded hash.
type PasswordHasher interface {
	// Hash returns a self-describing encoded hash of plain (salt and
	// parameters included).
	Hash(plain string) (string, error)
	// Verify reports whether plain matches hash. It returns an error only
	// when hash cannot be decoded, never for a wrong password.
	Verify(plain, hash string) (bool, error)
}

// ValidatePassword checks the password policy before hashing.
func ValidatePassword(plain string) error {
	n := utf8.RuneCountInString(plain)
	switch {
	case n < MinPasswordLength:
		return ErrPasswordTooShort
	case n > MaxPasswordLength:
		return ErrPasswordTooLong
	}
	return nil
}
