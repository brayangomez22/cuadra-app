package onboarding

import "github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"

// ValidateSteps checks a business template's getting-started steps against
// the catalog (see domain.ValidateSteps). internal/server runs it on every
// template at startup.
func ValidateSteps(ids []string) error { return domain.ValidateSteps(ids) }
