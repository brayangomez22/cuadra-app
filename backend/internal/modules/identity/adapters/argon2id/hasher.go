// Package argon2id implements domain.PasswordHasher with Argon2id (RFC 9106).
package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// ErrMalformedHash means a stored hash cannot be decoded as Argon2id PHC.
var ErrMalformedHash = errors.New("argon2id: malformed hash")

// Params are the Argon2id cost parameters. Memory is in KiB.
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams follow the OWASP Password Storage Cheat Sheet minimum for
// Argon2id: 19 MiB, 2 iterations, 1 lane.
var DefaultParams = Params{Memory: 19 * 1024, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32}

// Hasher hashes with its own params and verifies with the params encoded in
// each hash, so params can be raised without invalidating stored hashes.
type Hasher struct {
	params Params
}

var _ domain.PasswordHasher = (*Hasher)(nil)

// New returns a hasher that creates hashes with p.
func New(p Params) *Hasher { return &Hasher{params: p} }

var b64 = base64.RawStdEncoding

// Hash returns plain hashed in PHC string format:
// $argon2id$v=19$m=<memory>,t=<iterations>,p=<parallelism>$<salt>$<key>.
func (h *Hasher) Hash(plain string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2id: generate salt: %w", err)
	}
	p := h.params
	key := argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify recomputes the key with the hash's own salt and params and compares
// it in constant time.
func (h *Hasher) Verify(plain, hash string) (bool, error) {
	p, salt, key, err := decode(hash)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, uint32(len(key))) //nolint:gosec // len(key) comes from a decoded hash, far below MaxUint32
	return subtle.ConstantTimeCompare(got, key) == 1, nil
}

func decode(hash string) (p Params, salt, key []byte, err error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrMalformedHash
	}

	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		return Params{}, nil, nil, ErrMalformedHash
	}
	// Re-encoding the scanned params must give back the same text, which
	// rejects trailing garbage and non-canonical numbers.
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism)
	if err != nil || parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", p.Memory, p.Iterations, p.Parallelism) {
		return Params{}, nil, nil, ErrMalformedHash
	}
	if p.Memory == 0 || p.Iterations == 0 || p.Parallelism == 0 {
		return Params{}, nil, nil, ErrMalformedHash
	}

	salt, err = b64.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return Params{}, nil, nil, ErrMalformedHash
	}
	key, err = b64.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return Params{}, nil, nil, ErrMalformedHash
	}
	return p, salt, key, nil
}
