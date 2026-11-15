package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxTestSetLength = 100

// Mode is the provider's environment for a tenant's documents. A tenant
// starts in test, which has no legal effect; habilitation sends the DIAN's
// test set, the step that enables a business to issue; production issues
// documents with legal validity.
type Mode string

// The provider's modes.
const (
	ModeTest         Mode = "test"
	ModeHabilitation Mode = "habilitation"
	ModeProduction   Mode = "production"
)

// ParseMode returns the Mode of s, or ErrInvalidProviderMode.
func ParseMode(s string) (Mode, error) {
	switch m := Mode(s); m {
	case ModeTest, ModeHabilitation, ModeProduction:
		return m, nil
	}
	return "", ErrInvalidProviderMode
}

// ProviderAccount is the tenant's account at the technology provider: the
// token of its organization, the mode its documents go in and, during the
// habilitation, the test set the DIAN assigned it.
type ProviderAccount struct {
	Mode      Mode
	TestSetID string
	// Token is the organization's token; zero until the tenant sets it.
	Token Secret
}

// DefaultProviderAccount is the account of a tenant that never set one:
// test mode, without token.
func DefaultProviderAccount() ProviderAccount { return ProviderAccount{Mode: ModeTest} }

// NewProviderAccount validates an account: a known mode
// (ErrInvalidProviderMode) and a test set of one line of at most 100
// characters (ErrInvalidTestSet), required in habilitation
// (ErrMissingTestSet). The token may be zero.
func NewProviderAccount(mode, testSetID string, token Secret) (ProviderAccount, error) {
	m, err := ParseMode(mode)
	if err != nil {
		return ProviderAccount{}, err
	}
	testSetID = strings.TrimSpace(testSetID)
	if utf8.RuneCountInString(testSetID) > maxTestSetLength || strings.ContainsFunc(testSetID, unicode.IsControl) {
		return ProviderAccount{}, ErrInvalidTestSet
	}
	if m == ModeHabilitation && testSetID == "" {
		return ProviderAccount{}, ErrMissingTestSet
	}
	return ProviderAccount{Mode: m, TestSetID: testSetID, Token: token}, nil
}

// HasToken is whether the tenant set its organization's token.
func (a ProviderAccount) HasToken() bool { return !a.Token.IsZero() }
