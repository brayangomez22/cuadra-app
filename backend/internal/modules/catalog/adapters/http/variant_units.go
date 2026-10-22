package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// SetVariantUnit adds or replaces an alternate unit of a variant. The unit
// is a resource of the path: a missing one is a 404.
func (h *Handler) SetVariantUnit(ctx context.Context, req catalogapi.SetVariantUnitRequestObject) (catalogapi.SetVariantUnitResponseObject, error) {
	b := req.Body
	factor, err := parseAmount(b.Factor, domain.ErrInvalidUnitFactor)
	if err != nil {
		return nil, err
	}
	price, err := parseAmount(b.Price, domain.ErrInvalidPrice)
	if err != nil {
		return nil, err
	}
	in := app.VariantUnitInput{Factor: factor, Price: price}
	if b.Barcode != nil {
		in.Barcode = *b.Barcode
	}
	p, err := h.uc.SetVariantUnit(ctx, actor(ctx), req.ProductId, req.VariantId, req.UnitId, in)
	if err != nil {
		return nil, unitPathError(err)
	}
	return catalogapi.SetVariantUnit200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// RemoveVariantUnit removes an alternate unit of a variant.
func (h *Handler) RemoveVariantUnit(ctx context.Context, req catalogapi.RemoveVariantUnitRequestObject) (catalogapi.RemoveVariantUnitResponseObject, error) {
	p, err := h.uc.RemoveVariantUnit(ctx, actor(ctx), req.ProductId, req.VariantId, req.UnitId)
	if err != nil {
		return nil, err
	}
	return catalogapi.RemoveVariantUnit200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// variantUnitsBody builds a variant's alternate units of the response.
func variantUnitsBody(v *domain.ProductVariant, rate domain.TaxRate) []catalogapi.VariantUnit {
	units := v.Units()
	out := make([]catalogapi.VariantUnit, 0, len(units))
	for _, u := range units {
		body := catalogapi.VariantUnit{
			Unit:         unitBody(u.Unit()),
			Factor:       u.Factor().String(),
			Price:        formatMoney(u.Price()),
			PriceWithTax: formatMoney(u.PriceWithTax(rate)),
		}
		if barcode := u.Barcode(); barcode != "" {
			body.Barcode = &barcode
		}
		out = append(out, body)
	}
	return out
}
