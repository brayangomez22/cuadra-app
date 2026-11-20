package ports

import (
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/templates"
	"github.com/brayangomez22/cuadra-app/backend/templates/business"
)

// LoadTemplates loads the business templates embedded in the binary,
// checking their modules against registry and their getting-started steps
// against the onboarding module's catalog. It fails, so the API does not
// start, if a template is invalid.
func LoadTemplates(registry *modules.Registry) (*templates.Catalog, error) {
	return templates.Load(business.FS, templates.Checks{Modules: registry.CheckSet, Steps: onboarding.ValidateSteps})
}
