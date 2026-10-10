package domain

import (
	"context"

	"github.com/google/uuid"
)

// ProductRepository persists products. Every operation is scoped to a tenant:
// a product of another tenant is reported as ErrProductNotFound.
type ProductRepository interface {
	// Create stores a new product; ErrSKUTaken or ErrBarcodeTaken if another
	// product of the tenant uses them, ErrCategoryNotFound if its category
	// does not exist in the tenant.
	Create(ctx context.Context, p *Product) error
	// GetByID returns ErrProductNotFound when the product does not exist in tenantID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Product, error)
	// Update stores every field but the id, tenant and creation time, with
	// the errors of Create; ErrProductNotFound if it does not exist.
	Update(ctx context.Context, p *Product) error
	// Search returns one page of the tenant's products matching f.
	Search(ctx context.Context, tenantID uuid.UUID, f ProductFilter) (ProductPage, error)
}

// ProductFilter selects products. Zero values select everything: no text,
// any category, active or not.
type ProductFilter struct {
	// Query matches an exact SKU or barcode, or names containing each of its
	// words (case- and accent-insensitive).
	Query      string
	CategoryID *uuid.UUID
	// Active, when set, keeps only active (true) or inactive (false) products.
	Active *bool
	Limit  int
	Offset int
}

// ProductPage is one page of a search and how many products match in total.
type ProductPage struct {
	Items []*Product
	Total int
}

// CategoryRepository persists categories, scoped to a tenant like
// ProductRepository.
type CategoryRepository interface {
	// Create stores a new category; ErrCategoryNotFound if its parent does
	// not exist in the tenant.
	Create(ctx context.Context, c *Category) error
	// GetByID returns ErrCategoryNotFound when the category does not exist in tenantID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Category, error)
	// List returns every category of the tenant, by name.
	List(ctx context.Context, tenantID uuid.UUID) ([]*Category, error)
	// Update stores the name and parent; ErrCategoryNotFound if the category
	// or its parent does not exist.
	Update(ctx context.Context, c *Category) error
	// Delete removes the category; ErrCategoryInUse if it has products or
	// subcategories, ErrCategoryNotFound if it does not exist.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// Ancestors returns id and its ancestors, nearest first; none if id does
	// not exist in tenantID.
	Ancestors(ctx context.Context, tenantID, id uuid.UUID) ([]uuid.UUID, error)
}
