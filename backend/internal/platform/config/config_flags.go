package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// defaultFlagsCacheTTL is how long each instance reuses what it read of the
// feature flags: a change shows in every instance within it.
const defaultFlagsCacheTTL = 30 * time.Second

// loadFlags reads PLATFORM_OPERATORS (the IDs of the users of the team who
// manage the feature flags, comma-separated) and FEATURE_FLAGS_CACHE_TTL.
func (cfg *Config) loadFlags(getenv func(string) string) error {
	var errs []error
	for _, item := range list(getenv("PLATFORM_OPERATORS")) {
		id, err := uuid.Parse(item)
		if err != nil {
			errs = append(errs, fmt.Errorf("PLATFORM_OPERATORS: %q is not a user ID", item))
			continue
		}
		cfg.PlatformOperators = append(cfg.PlatformOperators, id)
	}
	cfg.FlagsCacheTTL = defaultFlagsCacheTTL
	if raw := getenv("FEATURE_FLAGS_CACHE_TTL"); raw != "" {
		d, err := time.ParseDuration(raw)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("FEATURE_FLAGS_CACHE_TTL is invalid: %w", err))
		case d <= 0:
			errs = append(errs, fmt.Errorf("FEATURE_FLAGS_CACHE_TTL must be positive, got %s", d))
		default:
			cfg.FlagsCacheTTL = d
		}
	}
	return errors.Join(errs...)
}
