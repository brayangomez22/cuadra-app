package ports

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing"
	onboardingapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/templates"
)

// OnboardingTemplates satisfies onboardingapp.Templates with the embedded
// business templates.
type OnboardingTemplates struct{ Catalog *templates.Catalog }

// Steps returns the getting-started steps of template id.
func (t OnboardingTemplates) Steps(id string) ([]string, bool) {
	tpl, ok := t.Catalog.Get(id)
	if !ok {
		return nil, false
	}
	return tpl.GettingStarted, true
}

// OnboardingInvoicing satisfies onboardingapp.Invoicing with the invoicing
// module's readiness.
type OnboardingInvoicing struct{ Invoicing *invoicing.Module }

// Readiness reads whether the tenant's electronic invoicing is ready.
func (i OnboardingInvoicing) Readiness(ctx context.Context, a onboardingapp.Actor) (onboardingapp.InvoicingReadiness, error) {
	r, err := i.Invoicing.Readiness(ctx, a.TenantID, a.UserID)
	if err != nil {
		return onboardingapp.InvoicingReadiness{}, err
	}
	return onboardingapp.InvoicingReadiness{Enabled: r.Enabled, Ready: r.Ready}, nil
}
