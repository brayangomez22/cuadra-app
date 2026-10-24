package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// VariantUseCases are the use cases of a product's variants and their
// alternate units.
type VariantUseCases interface {
	AddVariant(ctx context.Context, a app.Actor, productID uuid.UUID, in app.VariantInput) (*domain.Product, error)
	UpdateVariant(ctx context.Context, a app.Actor, productID, variantID uuid.UUID, in app.VariantUpdate) (*domain.Product, error)
	SetVariantUnit(ctx context.Context, a app.Actor, productID, variantID, unitID uuid.UUID, in app.VariantUnitInput) (*domain.Product, error)
	RemoveVariantUnit(ctx context.Context, a app.Actor, productID, variantID, unitID uuid.UUID) (*domain.Product, error)
}

// variantHandlers are the operations on variants and their units.
type variantHandlers struct {
	uc VariantUseCases
	costView
}

// AddProductVariant adds a variant to a product.
func (h *variantHandlers) AddProductVariant(ctx context.Context, req catalogapi.AddProductVariantRequestObject) (catalogapi.AddProductVariantResponseObject, error) {
	b := req.Body
	in, err := variantInput(b.Sku, b.Barcode, b.Cost, b.Price, b.AxisValues)
	if err != nil {
		return nil, err
	}
	p, err := h.uc.AddVariant(ctx, actor(ctx), req.ProductId, in)
	if err != nil {
		return nil, err
	}
	return catalogapi.AddProductVariant201JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// UpdateProductVariant replaces a variant's data and state.
func (h *variantHandlers) UpdateProductVariant(ctx context.Context, req catalogapi.UpdateProductVariantRequestObject) (catalogapi.UpdateProductVariantResponseObject, error) {
	b := req.Body
	in, err := variantInput(b.Sku, b.Barcode, b.Cost, b.Price, b.AxisValues)
	if err != nil {
		return nil, err
	}
	p, err := h.uc.UpdateVariant(ctx, actor(ctx), req.ProductId, req.VariantId, app.VariantUpdate{VariantInput: in, Active: b.Active})
	if err != nil {
		return nil, err
	}
	return catalogapi.UpdateProductVariant200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// variantInput translates a variant body. The contract already checked the
// formats; the business rules are the domain's.
func variantInput(sku string, barcode *string, costText, priceText string, axes catalogapi.AxisValues) (app.VariantInput, error) {
	cost, err := httpx.ParseDecimal(costText, domain.ErrInvalidCost)
	if err != nil {
		return app.VariantInput{}, err
	}
	price, err := httpx.ParseDecimal(priceText, domain.ErrInvalidPrice)
	if err != nil {
		return app.VariantInput{}, err
	}
	in := app.VariantInput{SKU: sku, Cost: cost, Price: price, AxisValues: axes}
	if barcode != nil {
		in.Barcode = *barcode
	}
	return in, nil
}

// variantBody builds a variant of the response; its price with tax uses the
// product's rate.
func variantBody(v *domain.ProductVariant, rate domain.TaxRate, seesCost bool) catalogapi.ProductVariant {
	body := catalogapi.ProductVariant{
		Id:           v.ID(),
		Sku:          v.SKU(),
		Price:        httpx.FormatMoney(v.Price()),
		PriceWithTax: httpx.FormatMoney(v.PriceWithTax(rate)),
		AxisValues:   v.AxisValues(),
		Label:        v.Label(),
		Units:        variantUnitsBody(v, rate),
		IsDefault:    v.IsDefault(),
		Active:       v.IsActive(),
		CreatedAt:    v.CreatedAt(),
	}
	if barcode := v.Barcode(); barcode != "" {
		body.Barcode = &barcode
	}
	if seesCost {
		cost := httpx.FormatMoney(v.Cost())
		margin := v.MarginPercent().String()
		body.Cost, body.MarginPercent = &cost, &margin
	}
	return body
}

// SetVariantUnit adds or replaces an alternate unit of a variant. The unit
// is a resource of the path: a missing one is a 404.
func (h *variantHandlers) SetVariantUnit(ctx context.Context, req catalogapi.SetVariantUnitRequestObject) (catalogapi.SetVariantUnitResponseObject, error) {
	b := req.Body
	factor, err := httpx.ParseDecimal(b.Factor, domain.ErrInvalidUnitFactor)
	if err != nil {
		return nil, err
	}
	price, err := httpx.ParseDecimal(b.Price, domain.ErrInvalidPrice)
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
func (h *variantHandlers) RemoveVariantUnit(ctx context.Context, req catalogapi.RemoveVariantUnitRequestObject) (catalogapi.RemoveVariantUnitResponseObject, error) {
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
			Price:        httpx.FormatMoney(u.Price()),
			PriceWithTax: httpx.FormatMoney(u.PriceWithTax(rate)),
		}
		if barcode := u.Barcode(); barcode != "" {
			body.Barcode = &barcode
		}
		out = append(out, body)
	}
	return out
}
