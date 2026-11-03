package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/http/salesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// SettingsUseCases are the use cases of the tenant's sales rules.
type SettingsUseCases interface {
	SalesSettings(ctx context.Context, a app.Actor) (domain.SalesPolicy, error)
	UpdateSalesSettings(ctx context.Context, a app.Actor, in app.SalesSettingsUpdate) (domain.SalesPolicy, error)
}

// GetSalesSettings returns the tenant's sales rules.
func (h *Handler) GetSalesSettings(ctx context.Context, _ salesapi.GetSalesSettingsRequestObject) (salesapi.GetSalesSettingsResponseObject, error) {
	policy, err := h.uc.SalesSettings(ctx, h.actor(ctx))
	if err != nil {
		return nil, err
	}
	return salesapi.GetSalesSettings200JSONResponse(salesSettings(policy)), nil
}

// UpdateSalesSettings changes the tenant's sales rules.
func (h *Handler) UpdateSalesSettings(ctx context.Context, req salesapi.UpdateSalesSettingsRequestObject) (salesapi.UpdateSalesSettingsResponseObject, error) {
	in := app.SalesSettingsUpdate{RequireCashSession: req.Body.RequireCashSession}
	if req.Body.MaxDiscount != nil {
		fraction, err := httpx.ParseDecimal(*req.Body.MaxDiscount, domain.ErrInvalidDiscountLimit)
		if err != nil {
			return nil, err
		}
		in.MaxDiscount = &fraction
	}
	policy, err := h.uc.UpdateSalesSettings(ctx, h.actor(ctx), in)
	if err != nil {
		return nil, err
	}
	return salesapi.UpdateSalesSettings200JSONResponse(salesSettings(policy)), nil
}

func salesSettings(p domain.SalesPolicy) salesapi.SalesSettings {
	return salesapi.SalesSettings{
		MaxDiscount:        httpx.FormatDecimal(p.MaxDiscount.Max()),
		RequireCashSession: !p.CashSessionOptional,
	}
}
