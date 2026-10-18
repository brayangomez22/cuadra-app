package domain

import (
	"context"

	"github.com/google/uuid"
)

// ProductRepository persists products. Every operation is scoped to a tenant:
// a product of another tenant is reported as ErrProductNotFound.
type ProductRepository interface {
	// Create stores a new product; ErrSKUTaken or ErrBarcodeTaken if another
	// product of the tenant uses them, ErrCategoryNotFound or ErrUnitNotFound
	// if its category or its base unit does not exist in the tenant.
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
	// IDs, when not nil, keeps only these products.
	IDs    []uuid.UUID
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

// UnitRepository persists units of measure, scoped to a tenant like
// ProductRepository.
type UnitRepository interface {
	// Create stores a new unit; ErrUnitCodeTaken if another unit of the
	// tenant has its code.
	Create(ctx context.Context, u *UnitOfMeasure) error
	// GetByID returns ErrUnitNotFound when the unit does not exist in tenantID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*UnitOfMeasure, error)
	// List returns every unit of the tenant, by name.
	List(ctx context.Context, tenantID uuid.UUID) ([]*UnitOfMeasure, error)
	// Update stores the code and the name; ErrUnitNotFound if the unit does
	// not exist, ErrUnitCodeTaken if another unit has the code.
	Update(ctx context.Context, u *UnitOfMeasure) error
	// Delete removes the unit; ErrUnitInUse if a product uses it,
	// ErrUnitNotFound if it does not exist.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// AttributeRepository persists attribute definitions, scoped to a tenant
// like ProductRepository.
type AttributeRepository interface {
	// Create stores a new definition; ErrAttributeKeyTaken if another
	// definition of the tenant has its key.
	Create(ctx context.Context, d *AttributeDefinition) error
	// GetByID returns ErrAttributeNotFound when the definition does not
	// exist in tenantID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*AttributeDefinition, error)
	// List returns every definition of the tenant, by label.
	List(ctx context.Context, tenantID uuid.UUID) ([]*AttributeDefinition, error)
	// Update stores the label, the options and whether it is required;
	// ErrAttributeNotFound if it does not exist, ErrAttributeOptionInUse
	// (and nothing changes) if a product has one of removedOptions.
	Update(ctx context.Context, d *AttributeDefinition, removedOptions []string) error
	// Delete removes the definition; ErrAttributeInUse if a product has a
	// value for it, ErrAttributeNotFound if it does not exist.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// VariantAxisRepository persists variant axes, scoped to a tenant like
// ProductRepository.
type VariantAxisRepository interface {
	// Create stores a new axis; ErrAxisNameTaken if another axis of the
	// tenant has its name.
	Create(ctx context.Context, a *VariantAxis) error
	// GetByID returns ErrAxisNotFound when the axis does not exist in tenantID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*VariantAxis, error)
	// List returns every axis of the tenant, by name.
	List(ctx context.Context, tenantID uuid.UUID) ([]*VariantAxis, error)
	// Update stores the allowed values; ErrAxisNotFound if the axis does not
	// exist, ErrAxisValueInUse (and nothing changes) if a variant has one of
	// removedValues, in any case.
	Update(ctx context.Context, a *VariantAxis, removedValues []string) error
	// Delete removes the axis; ErrAxisInUse if a variant has a value on it,
	// ErrAxisNotFound if it does not exist.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// TenantSeed is a new tenant's initial catalog configuration: what its
// business template declares, already validated by each constructor.
type TenantSeed struct {
	Units       []*UnitOfMeasure
	Categories  []*Category
	Attributes  []*AttributeDefinition
	VariantAxes []*VariantAxis
}

// SeedRepository stores a new tenant's initial catalog.
type SeedRepository interface {
	// Seed stores everything in seed for tenantID in one transaction, with
	// the errors of each repository's Create: either all of it is stored or
	// nothing is.
	Seed(ctx context.Context, tenantID uuid.UUID, seed TenantSeed) error
}
