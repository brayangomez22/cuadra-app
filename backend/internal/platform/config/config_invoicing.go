package config

import (
	"errors"
	"fmt"
)

// ProviderSimulated is the electronic invoicing provider that never reaches
// the DIAN. The real provider's adapter (Muisca) arrives with T36b.
const ProviderSimulated = "simulated"

// productionEnv is the APP_ENV of production.
const productionEnv = "production"

// loadInvoicing reads INVOICING_PROVIDER: the simulated provider by default
// outside production; in production it must be a real one, since a
// simulated document has no legal value and the sale would look invoiced.
func (cfg *Config) loadInvoicing(getenv func(string) string) error {
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
