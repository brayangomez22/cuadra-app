package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// CreateProductWithVariants adds a product with all its variants at once.
func (h *productHandlers) CreateProductWithVariants(ctx context.Context, req catalogapi.CreateProductWithVariantsRequestObject) (catalogapi.CreateProductWithVariantsResponseObject, error) {
	in, err := productWithVariantsInput(*req.Body)
	if err != nil {
		return nil, err
	}
	p, err := h.uc.CreateProductWithVariants(ctx, actor(ctx), in)
	if err != nil {
		return nil, err
	}
	return catalogapi.CreateProductWithVariants201JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// productWithVariantsInput translates the request body, like productInput.
func productWithVariantsInput(body catalogapi.ProductWithVariantsInput) (app.ProductWithVariantsInput, error) {
	rateValue, err := httpx.ParseDecimal(body.TaxRate, domain.ErrInvalidTaxRate)
	if err != nil {
		return app.ProductWithVariantsInput{}, err
	}
	rate, err := domain.ParseTaxRate(rateValue)
	if err != nil {
		return app.ProductWithVariantsInput{}, err
	}
	attrs, err := attributeValues(body.Attributes)
	if err != nil {
		return app.ProductWithVariantsInput{}, err
	}
	in := app.ProductWithVariantsInput{
		Name: body.Name, CategoryID: body.CategoryId, BaseUnitID: body.BaseUnitId,
		TaxRate: rate, Attributes: attrs,
	}
	if body.Description != nil {
		in.Description = *body.Description
	}
	for _, v := range body.Variants {
		variant, err := variantInput(v.Sku, v.Barcode, v.Cost, v.Price, v.AxisValues)
		if err != nil {
			return app.ProductWithVariantsInput{}, err
		}
		in.Variants = append(in.Variants, variant)
	}
	return in, nil
}
