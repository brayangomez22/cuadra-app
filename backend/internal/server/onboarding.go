package server

import (
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding"
	"github.com/brayangomez22/cuadra-app/backend/internal/server/ports"
)

// onboarding builds the getting-started checklist, which takes its steps
// from the business templates and asks the invoicing module whether
// electronic invoicing is ready.
func (w *wiring) onboarding(inv *invoicing.Module) (*onboarding.Module, error) {
	return onboarding.New(onboarding.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		Templates:      ports.OnboardingTemplates{Catalog: w.templates},
		Modules:        w.moduleService,
		Invoicing:      ports.OnboardingInvoicing{Invoicing: inv},
		Logger:         w.Log,
		Now:            w.Now,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
	})
}
