package config

import (
	"encoding/base64"
	"errors"
	"fmt"
)

// ProviderSimulated is the electronic invoicing provider that never reaches
// the DIAN. The real provider's adapter (Muisca) arrives with T36d.
const ProviderSimulated = "simulated"

// productionEnv is the APP_ENV of production.
const productionEnv = "production"

// loadInvoicing reads INVOICING_PROVIDER: the simulated provider by default
// outside production; in production it must be a real one, since a
// simulated document has no legal value and the sale would look invoiced.
func (cfg *Config) loadInvoicing(getenv func(string) string) error {
	if err := cfg.loadTokenKey(getenv); err != nil {
		return err
	}
	raw := getenv("INVOICING_PROVIDER")
	switch {
	case raw == "" && cfg.Env == productionEnv:
		return errors.New("INVOICING_PROVIDER is required in production")
	case raw == "":
		raw = ProviderSimulated
	case raw != ProviderSimulated:
		return fmt.Errorf("INVOICING_PROVIDER %q is unknown (valid: %s)", raw, ProviderSimulated)
	}
	if raw == ProviderSimulated && cfg.Env == productionEnv {
		return errors.New("INVOICING_PROVIDER cannot be simulated in production")
	}
	cfg.InvoicingProvider = raw
	return nil
}

// tokenKeySize is the size of INVOICING_TOKEN_KEY: an AES-256 key.
const tokenKeySize = 32

// loadTokenKey reads INVOICING_TOKEN_KEY, the key that encrypts the
// tenants' tokens at the provider: 32 bytes in base64, required in
// production. Outside production it may be empty, and the invoicing module
// uses its development key. An error never shows the value.
func (cfg *Config) loadTokenKey(getenv func(string) string) error {
	raw := getenv("INVOICING_TOKEN_KEY")
	if raw == "" {
		if cfg.Env == productionEnv {
			return errors.New("INVOICING_TOKEN_KEY is required in production")
		}
		return nil
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != tokenKeySize {
		return errors.New("INVOICING_TOKEN_KEY must be 32 bytes in base64 (openssl rand -base64 32)")
	}
	cfg.InvoicingTokenKey = key
	return nil
}
