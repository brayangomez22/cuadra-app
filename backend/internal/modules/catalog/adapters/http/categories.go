package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http/catalogapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// CategoryUseCases are the use cases of the categories.
type CategoryUseCases interface {
	CreateCategory(ctx context.Context, a app.Actor, name string, parentID *uuid.UUID) (*domain.Category, error)
	UpdateCategory(ctx context.Context, a app.Actor, id uuid.UUID, name string, parentID *uuid.UUID) (*domain.Category, error)
	ListCategories(ctx context.Context, a app.Actor) ([]*domain.Category, error)
	DeleteCategory(ctx context.Context, a app.Actor, id uuid.UUID) error
}

// categoryHandlers are the operations on categories.
type categoryHandlers struct{ uc CategoryUseCases }

// ListCategories returns every category of the tenant.
func (h *categoryHandlers) ListCategories(ctx context.Context, _ catalogapi.ListCategoriesRequestObject) (catalogapi.ListCategoriesResponseObject, error) {
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
func (h *categoryHandlers) CreateCategory(ctx context.Context, req catalogapi.CreateCategoryRequestObject) (catalogapi.CreateCategoryResponseObject, error) {
	c, err := h.uc.CreateCategory(ctx, actor(ctx), req.Body.Name, req.Body.ParentId)
	if err != nil {
		return nil, err
	}
	return catalogapi.CreateCategory201JSONResponse(categoryBody(c)), nil
}

// UpdateCategory renames and moves a category.
func (h *categoryHandlers) UpdateCategory(ctx context.Context, req catalogapi.UpdateCategoryRequestObject) (catalogapi.UpdateCategoryResponseObject, error) {
	c, err := h.uc.UpdateCategory(ctx, actor(ctx), req.CategoryId, req.Body.Name, req.Body.ParentId)
	if err != nil {
		return nil, err
	}
	return catalogapi.UpdateCategory200JSONResponse(categoryBody(c)), nil
}

// DeleteCategory removes an empty category.
func (h *categoryHandlers) DeleteCategory(ctx context.Context, req catalogapi.DeleteCategoryRequestObject) (catalogapi.DeleteCategoryResponseObject, error) {
	if err := h.uc.DeleteCategory(ctx, actor(ctx), req.CategoryId); err != nil {
		return nil, err
	}
	return catalogapi.DeleteCategory204Response{}, nil
}

func categoryBody(c *domain.Category) catalogapi.Category {
	return catalogapi.Category{Id: c.ID(), Name: c.Name(), ParentId: c.ParentID(), CreatedAt: c.CreatedAt()}
}
