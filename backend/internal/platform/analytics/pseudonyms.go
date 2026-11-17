// Package analytics holds what the backend gives the usage analytics of the
// app (T40): the pseudonymous ID of each user. The events themselves are
// sent by the frontend, and only when /me brings that ID.
package analytics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

// pseudonymLength is the hex characters kept of the HMAC: 128 bits.
const pseudonymLength = 32

// Pseudonyms turns user IDs into the IDs the analytics sees: an HMAC-SHA256
// with a key of the server (ANALYTICS_SECRET), so the analytics provider
// cannot tie them back to a user and rotating the key unlinks the history.
type Pseudonyms struct{ key []byte }

// NewPseudonyms builds Pseudonyms with key; without one the analytics is off.
func NewPseudonyms(key []byte) *Pseudonyms { return &Pseudonyms{key: key} }

// Pseudonym returns the user's pseudonym; ok is false when there is no key.
func (p *Pseudonyms) Pseudonym(userID uuid.UUID) (string, bool) {
	if len(p.key) == 0 {
		return "", false
	}
	mac := hmac.New(sha256.New, p.key)
	mac.Write(userID[:])
	return hex.EncodeToString(mac.Sum(nil))[:pseudonymLength], true
}
