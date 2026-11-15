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
type TokenCipher struct{ aead cipher.AEAD }

// NewTokenCipher builds the cipher of a 32 byte key.
func NewTokenCipher(key []byte) (TokenCipher, error) {
	if len(key) != TokenKeySize {
		return TokenCipher{}, errors.New("invoicing: the token key must have 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return TokenCipher{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return TokenCipher{}, err
	}
	return TokenCipher{aead: aead}, nil
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

// Open decrypts what Seal encrypted for the same tenant with the same key.
func (c TokenCipher) Open(tenantID uuid.UUID, sealed []byte) (string, error) {
	n := c.aead.NonceSize()
	if len(sealed) < n+c.aead.Overhead() {
		return "", errors.New("invoicing: stored token is too short")
	}
	plain, err := c.aead.Open(nil, sealed[:n], sealed[n:], tenantID[:])
	if err != nil {
		return "", errors.New("invoicing: stored token does not open with this key")
	}
	return string(plain), nil
}
