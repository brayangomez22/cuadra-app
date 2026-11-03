package httpadapter

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http/identityapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// UpdateTenant changes the settings of the user's tenant.
func (h *Handler) UpdateTenant(ctx context.Context, req identityapi.UpdateTenantRequestObject) (identityapi.UpdateTenantResponseObject, error) {
	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		return identityapi.UpdateTenant401JSONResponse{UnauthorizedJSONResponse: identityapi.UnauthorizedJSONResponse(errorBody("unauthorized", unauthorizedMessage))}, nil
	}
	in := app.TenantUpdate{Address: req.Body.Address, City: req.Body.City, Phone: req.Body.Phone}
	if req.Body.DefaultTaxRate != nil {
		rate, err := decimal.NewFromString(*req.Body.DefaultTaxRate)
		if err != nil {
			return nil, domain.ErrInvalidDefaultTaxRate
		}
		in.DefaultTaxRate = &rate
	}
	tenant, err := h.uc.UpdateTenant(ctx, p.TenantID, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return identityapi.UpdateTenant200JSONResponse{
		Id:               tenant.ID(),
		Name:             tenant.Name(),
		Nit:              tenant.NIT().String(),
		BusinessTemplate: tenant.BusinessTemplate(),
		DefaultTaxRate:   tenant.DefaultTaxRate().String(),
		Address:          tenant.Contact().Address(),
		City:             tenant.Contact().City(),
		Phone:            tenant.Contact().Phone(),
	}, nil
}
