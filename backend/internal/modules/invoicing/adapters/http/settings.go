package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/http/invoicingapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// SettingsUseCases are the use cases of the tenant's emission settings.
type SettingsUseCases interface {
	GetSettings(ctx context.Context, a app.Actor) (domain.Settings, error)
	UpdateSettings(ctx context.Context, a app.Actor, enabled bool, kind string) (domain.Settings, error)
}

// settingsHandlers are the operations on the emission settings.
type settingsHandlers struct{ uc SettingsUseCases }

// GetInvoicingSettings returns the tenant's settings.
func (h *settingsHandlers) GetInvoicingSettings(ctx context.Context, _ invoicingapi.GetInvoicingSettingsRequestObject) (invoicingapi.GetInvoicingSettingsResponseObject, error) {
	s, err := h.uc.GetSettings(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	return invoicingapi.GetInvoicingSettings200JSONResponse(settingsBody(s)), nil
}

// UpdateInvoicingSettings changes the tenant's settings.
func (h *settingsHandlers) UpdateInvoicingSettings(ctx context.Context, req invoicingapi.UpdateInvoicingSettingsRequestObject) (invoicingapi.UpdateInvoicingSettingsResponseObject, error) {
	s, err := h.uc.UpdateSettings(ctx, actor(ctx), req.Body.Enabled, string(req.Body.DefaultKind))
	if err != nil {
		return nil, err
	}
	return invoicingapi.UpdateInvoicingSettings200JSONResponse(settingsBody(s)), nil
}

func settingsBody(s domain.Settings) invoicingapi.InvoicingSettings {
	return invoicingapi.InvoicingSettings{Enabled: s.Enabled, DefaultKind: invoicingapi.ElectronicDocumentKind(s.DefaultKind)}
}
