package httpadapter

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// ProductUseCases are the use cases of the products.
type ProductUseCases interface {
	CreateProduct(ctx context.Context, a app.Actor, in app.ProductInput) (*domain.Product, error)
	UpdateProduct(ctx context.Context, a app.Actor, id uuid.UUID, in app.ProductInput) (*domain.Product, error)
	GetProduct(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error)
	SearchProducts(ctx context.Context, a app.Actor, f domain.ProductFilter) (domain.ProductPage, error)
	DeactivateProduct(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error)
	ActivateProduct(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error)
}

// productHandlers are the operations on products.
type productHandlers struct {
	uc ProductUseCases
	costView
}

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// opPOSSearch is the operation the POS latency SLO measures for a search
// with text (docs/slo.md).
const opPOSSearch = "pos.search"

// ListProducts searches the tenant's products.
func (h *productHandlers) ListProducts(ctx context.Context, req catalogapi.ListProductsRequestObject) (catalogapi.ListProductsResponseObject, error) {
	params := req.Params
	f := domain.ProductFilter{CategoryID: params.CategoryId, Limit: defaultPageSize}
	if params.Q != nil {
		f.Query = *params.Q
	}
	if strings.TrimSpace(f.Query) != "" {
		// A typed text or a scanned code: the cashier is waiting for it.
		httpx.SetOperation(ctx, opPOSSearch)
	}
	if params.Ids != nil {
		f.IDs = *params.Ids
	}
	if params.Limit != nil {
		f.Limit = *params.Limit
	}
	if params.Offset != nil {
		f.Offset = *params.Offset
	}
	status := catalogapi.Active
	if params.Status != nil {
		status = *params.Status
	}
	if status != catalogapi.All {
		active := status == catalogapi.Active
		f.Active = &active
	}

	page, err := h.uc.SearchProducts(ctx, actor(ctx), f)
	if err != nil {
		return nil, err
	}
	seesCost := h.seesCost(ctx)
	items := make([]catalogapi.Product, 0, len(page.Items))
	for _, p := range page.Items {
		items = append(items, productBody(p, seesCost))
	}
	return catalogapi.ListProducts200JSONResponse{Items: items, Total: page.Total}, nil
}

// CreateProduct adds a product to the catalog.
func (h *productHandlers) CreateProduct(ctx context.Context, req catalogapi.CreateProductRequestObject) (catalogapi.CreateProductResponseObject, error) {
	in, err := productInput(*req.Body)
	if err != nil {
		return nil, err
	}
	p, err := h.uc.CreateProduct(ctx, actor(ctx), in)
	if err != nil {
		return nil, err
	}
	return catalogapi.CreateProduct201JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// GetProduct returns a product, active or not.
func (h *productHandlers) GetProduct(ctx context.Context, req catalogapi.GetProductRequestObject) (catalogapi.GetProductResponseObject, error) {
	p, err := h.uc.GetProduct(ctx, actor(ctx), req.ProductId)
	if err != nil {
		return nil, err
	}
	return catalogapi.GetProduct200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// UpdateProduct replaces a product's data.
func (h *productHandlers) UpdateProduct(ctx context.Context, req catalogapi.UpdateProductRequestObject) (catalogapi.UpdateProductResponseObject, error) {
	in, err := productInput(*req.Body)
	if err != nil {
		return nil, err
	}
	p, err := h.uc.UpdateProduct(ctx, actor(ctx), req.ProductId, in)
	if err != nil {
		return nil, err
	}
	return catalogapi.UpdateProduct200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// ActivateProduct makes a product sellable again.
func (h *productHandlers) ActivateProduct(ctx context.Context, req catalogapi.ActivateProductRequestObject) (catalogapi.ActivateProductResponseObject, error) {
	p, err := h.uc.ActivateProduct(ctx, actor(ctx), req.ProductId)
	if err != nil {
		return nil, err
	}
	return catalogapi.ActivateProduct200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// DeactivateProduct hides a product from sales.
func (h *productHandlers) DeactivateProduct(ctx context.Context, req catalogapi.DeactivateProductRequestObject) (catalogapi.DeactivateProductResponseObject, error) {
	p, err := h.uc.DeactivateProduct(ctx, actor(ctx), req.ProductId)
	if err != nil {
		return nil, err
	}
	return catalogapi.DeactivateProduct200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// productInput translates the request body. The contract already checked the
// formats; the business rules are the domain's.
func productInput(body catalogapi.ProductInput) (app.ProductInput, error) {
	cost, err := httpx.ParseDecimal(body.Cost, domain.ErrInvalidCost)
	if err != nil {
		return app.ProductInput{}, err
	}
	price, err := httpx.ParseDecimal(body.Price, domain.ErrInvalidPrice)
	if err != nil {
		return app.ProductInput{}, err
	}
	rateValue, err := httpx.ParseDecimal(body.TaxRate, domain.ErrInvalidTaxRate)
	if err != nil {
		return app.ProductInput{}, err
	}
	rate, err := domain.ParseTaxRate(rateValue)
	if err != nil {
		return app.ProductInput{}, err
	}
	attrs, err := attributeValues(body.Attributes)
	if err != nil {
		return app.ProductInput{}, err
	}
	in := app.ProductInput{
		SKU:        body.Sku,
		Name:       body.Name,
		CategoryID: body.CategoryId,
		BaseUnitID: body.BaseUnitId,
		Cost:       cost,
		Price:      price,
		TaxRate:    rate,
		Attributes: attrs,
	}
	if body.Barcode != nil {
		in.Barcode = *body.Barcode
	}
	if body.Description != nil {
		in.Description = *body.Description
	}
	return in, nil
}

// productBody builds the response; the variants carry cost and margin only
// for who may edit the catalog.
func productBody(p *domain.Product, seesCost bool) catalogapi.Product {
	variants := make([]catalogapi.ProductVariant, 0, len(p.Variants()))
	for _, v := range p.Variants() {
		variants = append(variants, variantBody(v, p.TaxRate(), seesCost))
	}
	body := catalogapi.Product{
		Id:         p.ID(),
		Name:       p.Name(),
		CategoryId: p.CategoryID(),
		BaseUnitId: p.BaseUnit().ID,
		Unit:       unitBody(p.BaseUnit()),
		TaxRate:    formatRate(p.TaxRate()),
		Attributes: attributesBody(p.Attributes()),
		Active:     p.IsActive(),
		CreatedAt:  p.CreatedAt(),
		Variants:   variants,
	}
	if description := p.Description(); description != "" {
		body.Description = &description
	}
	return body
}

// formatRate writes a tax rate without trailing zeros: "0.19", "0".
func formatRate(r domain.TaxRate) string { return r.Rate().String() }
