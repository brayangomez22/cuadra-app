package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/http/invoicingapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// ProviderUseCases are the use cases of the tenant's account at the
// technology provider.
type ProviderUseCases interface {
	ProviderAccount(ctx context.Context, a app.Actor) (domain.ProviderAccount, error)
	UpdateProviderAccount(ctx context.Context, a app.Actor, in app.ProviderAccountInput) (domain.ProviderAccount, error)
}

// providerHandlers are the operations on the account at the provider. The
// token goes in, never out.
type providerHandlers struct{ uc ProviderUseCases }

// GetInvoicingProvider returns the tenant's account, without its token.
func (h *providerHandlers) GetInvoicingProvider(ctx context.Context, _ invoicingapi.GetInvoicingProviderRequestObject) (invoicingapi.GetInvoicingProviderResponseObject, error) {
	a, err := h.uc.ProviderAccount(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	return invoicingapi.GetInvoicingProvider200JSONResponse(providerBody(a)), nil
}

// UpdateInvoicingProvider changes the tenant's account.
func (h *providerHandlers) UpdateInvoicingProvider(ctx context.Context, req invoicingapi.UpdateInvoicingProviderRequestObject) (invoicingapi.UpdateInvoicingProviderResponseObject, error) {
	in := app.ProviderAccountInput{Mode: string(req.Body.Mode), Token: req.Body.Token}
	if req.Body.TestSetId != nil {
		in.TestSetID = *req.Body.TestSetId
	}
	a, err := h.uc.UpdateProviderAccount(ctx, actor(ctx), in)
	if err != nil {
		return nil, err
	}
	return invoicingapi.UpdateInvoicingProvider200JSONResponse(providerBody(a)), nil
}

func providerBody(a domain.ProviderAccount) invoicingapi.InvoicingProvider {
	return invoicingapi.InvoicingProvider{
		Mode: invoicingapi.InvoicingProviderMode(a.Mode), TestSetId: a.TestSetID, HasToken: a.HasToken(),
	}
}
