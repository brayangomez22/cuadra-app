package postgres

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"

	"github.com/google/uuid"
)

// TokenKeySize is the size of the key of the provider's tokens: AES-256.
const TokenKeySize = 32

// TokenCipher encrypts the tenants' tokens at the provider with AES-256-GCM
// before they reach the database. The tenant's id is the additional data, so
// a ciphertext copied to another tenant's row does not open. The key comes
// from the environment (INVOICING_TOKEN_KEY), never from the database.
//
// During a rotation the previous key (INVOICING_TOKEN_KEY_PREVIOUS) still
// opens the tokens it encrypted (OpenRotating), and the repository encrypts
// them again with the current one (P167).
type TokenCipher struct {
	aead     cipher.AEAD
	previous cipher.AEAD
}

// NewTokenCipher builds the cipher of a 32 byte key and, during a
// rotation, of the previous one.
func NewTokenCipher(key []byte, previous ...[]byte) (TokenCipher, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return TokenCipher{}, err
	}
	c := TokenCipher{aead: aead}
	if len(previous) > 0 && previous[0] != nil {
		if c.previous, err = newAEAD(previous[0]); err != nil {
			return TokenCipher{}, err
		}
	}
	return c, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != TokenKeySize {
		return nil, errors.New("invoicing: the token key must have 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Seal encrypts the tenant's token: a random nonce followed by the
// ciphertext.
func (c TokenCipher) Seal(tenantID uuid.UUID, token string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize(), c.aead.NonceSize()+len(token)+c.aead.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, []byte(token), tenantID[:]), nil
}

// Open decrypts what Seal encrypted for the same tenant with the current
// key.
func (c TokenCipher) Open(tenantID uuid.UUID, sealed []byte) (string, error) {
	return open(c.aead, tenantID, sealed)
}

// OpenRotating decrypts with the current key or, failing that, with the
// previous one; stale tells that the token must be encrypted again.
func (c TokenCipher) OpenRotating(tenantID uuid.UUID, sealed []byte) (plain string, stale bool, err error) {
	plain, err = open(c.aead, tenantID, sealed)
	if err == nil || c.previous == nil {
		return plain, false, err
	}
	plain, err = open(c.previous, tenantID, sealed)
	return plain, err == nil, err
}

func open(aead cipher.AEAD, tenantID uuid.UUID, sealed []byte) (string, error) {
	n := aead.NonceSize()
	if len(sealed) < n+aead.Overhead() {
		return "", errors.New("invoicing: stored token is too short")
	}
	plain, err := aead.Open(nil, sealed[:n], sealed[n:], tenantID[:])
	if err != nil {
		return "", errors.New("invoicing: stored token does not open with this key")
	}
	return string(plain), nil
}
