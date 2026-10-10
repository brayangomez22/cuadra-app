// Package httpadapter exposes the catalog use cases over HTTP: it implements
// the strict server generated from the "catalog" operations of
// api/openapi.yaml (catalogapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// permissionCatalogWrite lets a role see a product's cost and margin. It is
// the identity module's "catalog:write", named here as the contract names it.
const permissionCatalogWrite = "catalog:write"

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// UseCases are the catalog use cases the handlers call (*app.Service).
type UseCases interface {
	CreateProduct(ctx context.Context, a app.Actor, in app.ProductInput) (*domain.Product, error)
	UpdateProduct(ctx context.Context, a app.Actor, id uuid.UUID, in app.ProductInput) (*domain.Product, error)
	GetProduct(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error)
	SearchProducts(ctx context.Context, a app.Actor, f domain.ProductFilter) (domain.ProductPage, error)
	DeactivateProduct(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error)
	ActivateProduct(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error)
	CreateCategory(ctx context.Context, a app.Actor, name string, parentID *uuid.UUID) (*domain.Category, error)
	UpdateCategory(ctx context.Context, a app.Actor, id uuid.UUID, name string, parentID *uuid.UUID) (*domain.Category, error)
	ListCategories(ctx context.Context, a app.Actor) ([]*domain.Category, error)
	DeleteCategory(ctx context.Context, a app.Actor, id uuid.UUID) error
}

// Handler implements catalogapi.StrictServerInterface.
type Handler struct {
	uc  UseCases
	can auth.Authorizer
}

var _ catalogapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the catalog operations on mux, with the module's error
// mapping. can decides who sees a product's cost and margin.
func Register(mux *http.ServeMux, uc UseCases, can auth.Authorizer, log *slog.Logger) {
	h := &Handler{uc: uc, can: can}
	errs := httpx.NewErrorMapper(log, errorRules...)
	catalogapi.HandlerWithOptions(
		catalogapi.NewStrictHandlerWithOptions(h, []catalogapi.StrictMiddlewareFunc{requirePrincipal}, catalogapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		catalogapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// requirePrincipal answers 401 when no principal reached the handlers. The
// API's auth middleware already demands one; this keeps a wiring mistake
// from running a use case without a tenant.
func requirePrincipal(next catalogapi.StrictHandlerFunc, _ string) catalogapi.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
		if _, ok := auth.PrincipalFrom(ctx); !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", unauthorizedMessage)
			return nil, nil
		}
		return next(ctx, w, r, req)
	}
}

// actor returns who calls: tenant and user come only from the access token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}
}

// seesCost reports whether the caller may see costs and margins.
func (h *Handler) seesCost(ctx context.Context) bool {
	p, _ := auth.PrincipalFrom(ctx)
	return h.can(p.Role, permissionCatalogWrite)
}

// ListProducts searches the tenant's products.
func (h *Handler) ListProducts(ctx context.Context, req catalogapi.ListProductsRequestObject) (catalogapi.ListProductsResponseObject, error) {
	params := req.Params
	f := domain.ProductFilter{CategoryID: params.CategoryId, Limit: defaultPageSize}
	if params.Q != nil {
		f.Query = *params.Q
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
func (h *Handler) CreateProduct(ctx context.Context, req catalogapi.CreateProductRequestObject) (catalogapi.CreateProductResponseObject, error) {
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
func (h *Handler) GetProduct(ctx context.Context, req catalogapi.GetProductRequestObject) (catalogapi.GetProductResponseObject, error) {
	p, err := h.uc.GetProduct(ctx, actor(ctx), req.ProductId)
	if err != nil {
		return nil, err
	}
	return catalogapi.GetProduct200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// UpdateProduct replaces a product's data.
func (h *Handler) UpdateProduct(ctx context.Context, req catalogapi.UpdateProductRequestObject) (catalogapi.UpdateProductResponseObject, error) {
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
func (h *Handler) ActivateProduct(ctx context.Context, req catalogapi.ActivateProductRequestObject) (catalogapi.ActivateProductResponseObject, error) {
	p, err := h.uc.ActivateProduct(ctx, actor(ctx), req.ProductId)
	if err != nil {
		return nil, err
	}
	return catalogapi.ActivateProduct200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// DeactivateProduct hides a product from sales.
func (h *Handler) DeactivateProduct(ctx context.Context, req catalogapi.DeactivateProductRequestObject) (catalogapi.DeactivateProductResponseObject, error) {
	p, err := h.uc.DeactivateProduct(ctx, actor(ctx), req.ProductId)
	if err != nil {
		return nil, err
	}
	return catalogapi.DeactivateProduct200JSONResponse(productBody(p, h.seesCost(ctx))), nil
}

// ListCategories returns every category of the tenant.
func (h *Handler) ListCategories(ctx context.Context, _ catalogapi.ListCategoriesRequestObject) (catalogapi.ListCategoriesResponseObject, error) {
	list, err := h.uc.ListCategories(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]catalogapi.Category, 0, len(list))
	for _, c := range list {
		items = append(items, categoryBody(c))
	}
	return catalogapi.ListCategories200JSONResponse{Items: items}, nil
}

// CreateCategory adds a category.
func (h *Handler) CreateCategory(ctx context.Context, req catalogapi.CreateCategoryRequestObject) (catalogapi.CreateCategoryResponseObject, error) {
	c, err := h.uc.CreateCategory(ctx, actor(ctx), req.Body.Name, req.Body.ParentId)
	if err != nil {
		return nil, err
	}
	return catalogapi.CreateCategory201JSONResponse(categoryBody(c)), nil
}

// UpdateCategory renames and moves a category.
func (h *Handler) UpdateCategory(ctx context.Context, req catalogapi.UpdateCategoryRequestObject) (catalogapi.UpdateCategoryResponseObject, error) {
	c, err := h.uc.UpdateCategory(ctx, actor(ctx), req.CategoryId, req.Body.Name, req.Body.ParentId)
	if err != nil {
		return nil, err
	}
	return catalogapi.UpdateCategory200JSONResponse(categoryBody(c)), nil
}

// DeleteCategory removes an empty category.
func (h *Handler) DeleteCategory(ctx context.Context, req catalogapi.DeleteCategoryRequestObject) (catalogapi.DeleteCategoryResponseObject, error) {
	if err := h.uc.DeleteCategory(ctx, actor(ctx), req.CategoryId); err != nil {
		return nil, err
	}
	return catalogapi.DeleteCategory204Response{}, nil
}

// productInput translates the request body. The contract already checked the
// formats; the business rules are the domain's.
func productInput(body catalogapi.ProductInput) (app.ProductInput, error) {
	cost, err := parseAmount(body.Cost, domain.ErrInvalidCost)
	if err != nil {
		return app.ProductInput{}, err
	}
	price, err := parseAmount(body.Price, domain.ErrInvalidPrice)
	if err != nil {
		return app.ProductInput{}, err
	}
	rateValue, err := parseAmount(body.TaxRate, domain.ErrInvalidTaxRate)
	if err != nil {
		return app.ProductInput{}, err
	}
	rate, err := domain.ParseTaxRate(rateValue)
	if err != nil {
		return app.ProductInput{}, err
	}
	unit, err := domain.ParseUnit(string(body.BaseUnit))
	if err != nil {
		return app.ProductInput{}, err
	}
	in := app.ProductInput{
		SKU:        body.Sku,
		Name:       body.Name,
		CategoryID: body.CategoryId,
		BaseUnit:   unit,
		Cost:       cost,
		Price:      price,
		TaxRate:    rate,
	}
	if body.Barcode != nil {
		in.Barcode = *body.Barcode
	}
	if body.Description != nil {
		in.Description = *body.Description
	}
	return in, nil
}

// productBody builds the response; cost and margin only for who may edit
// the catalog.
func productBody(p *domain.Product, seesCost bool) catalogapi.Product {
	body := catalogapi.Product{
		Id:           p.ID(),
		Sku:          p.SKU(),
		Name:         p.Name(),
		CategoryId:   p.CategoryID(),
		BaseUnit:     catalogapi.UnitOfMeasure(p.BaseUnit()),
		Price:        formatMoney(p.Price()),
		PriceWithTax: formatMoney(p.PriceWithTax()),
		TaxRate:      formatRate(p.TaxRate()),
		Active:       p.IsActive(),
		CreatedAt:    p.CreatedAt(),
	}
	if barcode := p.Barcode(); barcode != "" {
		body.Barcode = &barcode
	}
	if description := p.Description(); description != "" {
		body.Description = &description
	}
	if seesCost {
		cost := formatMoney(p.Cost())
		margin := p.MarginPercent().String()
		body.Cost, body.MarginPercent = &cost, &margin
	}
	return body
}

func categoryBody(c *domain.Category) catalogapi.Category {
	return catalogapi.Category{Id: c.ID(), Name: c.Name(), ParentId: c.ParentID(), CreatedAt: c.CreatedAt()}
}
