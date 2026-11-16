package config

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// HSTS tells whether the API asks browsers to use only HTTPS: in
// production, which is always behind TLS.
func (cfg Config) HSTS() bool { return cfg.Env == productionEnv }

// loadSecurity reads the settings of the API's edge and of the key
// rotations: TRUSTED_PROXIES, CORS_ALLOWED_ORIGINS, JWT_SECRET_PREVIOUS and
// INVOICING_TOKEN_KEY_PREVIOUS. An error never shows a key.
func (cfg *Config) loadSecurity(getenv func(string) string) error {
	var errs []error
	var err error
	if cfg.TrustedProxies, err = trustedProxies(getenv("TRUSTED_PROXIES")); err != nil {
		errs = append(errs, err)
	}
	if cfg.CORSAllowedOrigins, err = allowedOrigins(getenv("CORS_ALLOWED_ORIGINS")); err != nil {
		errs = append(errs, err)
	}
	if err := cfg.loadPreviousKeys(getenv); err != nil {
		errs = append(errs, err)
	}
	if err := cfg.loadRateLimits(getenv("RATE_LIMITS_ENABLED")); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// loadRateLimits reads RATE_LIMITS_ENABLED: on unless it says otherwise,
// and always on in production.
func (cfg *Config) loadRateLimits(raw string) error {
	cfg.RateLimitsEnabled = true
	if raw == "" {
		return nil
	}
	on, err := strconv.ParseBool(raw)
	switch {
	case err != nil:
		return fmt.Errorf("RATE_LIMITS_ENABLED is invalid: %w", err)
	case !on && cfg.Env == productionEnv:
		return errors.New("RATE_LIMITS_ENABLED cannot be off in production")
	}
	cfg.RateLimitsEnabled = on
	return nil
}

// list splits a comma-separated variable, without empty items.
func list(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// trustedProxies reads CIDR ranges or single IPs.
func trustedProxies(raw string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, item := range list(raw) {
		if p, err := netip.ParsePrefix(item); err == nil {
			out = append(out, p.Masked())
			continue
		}
		addr, err := netip.ParseAddr(item)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES: %q is not an IP nor a CIDR range", item)
		}
		out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return out, nil
}

// allowedOrigins reads origins: scheme and host, without path nor wildcard.
func allowedOrigins(raw string) ([]string, error) {
	var out []string
	for _, item := range list(raw) {
		u, err := url.Parse(item)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" ||
			u.Path != "" || u.RawQuery != "" || strings.Contains(u.Host, "*") {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS: %q is not an origin like https://app.example.com", item)
		}
		out = append(out, item)
	}
	return out, nil
}

// loadPreviousKeys reads the keys a rotation keeps for a while: the JWT
// secret's (as long as the access tokens it signed live) and the token
// key's (until every stored token is encrypted again).
func (cfg *Config) loadPreviousKeys(getenv func(string) string) error {
	var errs []error
	if raw := getenv("JWT_SECRET_PREVIOUS"); raw != "" {
		switch {
		case len(raw) < minJWTSecretLength:
			errs = append(errs, fmt.Errorf("JWT_SECRET_PREVIOUS must have at least %d bytes", minJWTSecretLength))
		case bytes.Equal([]byte(raw), cfg.JWTSecret):
			errs = append(errs, errors.New("JWT_SECRET_PREVIOUS must differ from JWT_SECRET"))
		default:
			cfg.JWTPreviousSecret = []byte(raw)
		}
	}
	if raw := getenv("INVOICING_TOKEN_KEY_PREVIOUS"); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		switch {
		case cfg.InvoicingTokenKey == nil:
			errs = append(errs, errors.New("INVOICING_TOKEN_KEY_PREVIOUS requires INVOICING_TOKEN_KEY"))
		case err != nil || len(key) != tokenKeySize:
			errs = append(errs, errors.New("INVOICING_TOKEN_KEY_PREVIOUS must be 32 bytes in base64"))
		default:
			cfg.InvoicingPreviousTokenKey = key
		}
	}
	return errors.Join(errs...)
}
