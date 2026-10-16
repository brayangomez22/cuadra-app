package domain

import (
	"context"

	"github.com/google/uuid"
)

// LocationRepository persists locations. Every operation is scoped to a
// tenant: a location of another tenant is reported as ErrLocationNotFound.
type LocationRepository interface {
	// Create stores a new location; ErrLocationNameTaken if another location
	// of the tenant has the same name, ignoring case.
	Create(ctx context.Context, l *Location) error
	// GetByID returns ErrLocationNotFound when the location does not exist in tenantID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Location, error)
	// List returns every location of the tenant, by name.
	List(ctx context.Context, tenantID uuid.UUID) ([]*Location, error)
}

// PolicyRepository reads the tenant's stock rules.
type PolicyRepository interface {
	// Get returns the tenant's policy, or the zero StockPolicy if it has
	// not set one.
	Get(ctx context.Context, tenantID uuid.UUID) (StockPolicy, error)
}

// StockChange is the operation Apply runs on the locked levels: it changes
// them through their methods and returns the movements that record it.
type StockChange func(levels []*StockLevel) ([]*StockMovement, error)

// StockRepository persists stock levels and the kardex.
type StockRepository interface {
	// Apply locks the levels of variantID at locationIDs (distinct), starting
	// from empty levels where there is none, and calls change with them in
	// the order of locationIDs. If change succeeds, it stores the levels and
	// the movements in the same transaction; otherwise it stores nothing and
	// returns change's error. ErrVariantNotFound or ErrLocationNotFound if the
	// variant or a location does not exist in tenantID. It works on the stock
	// without a lot (lot_id NULL).
	Apply(ctx context.Context, tenantID, variantID uuid.UUID, locationIDs []uuid.UUID, change StockChange) error
	// ListLevels returns one page of the levels at a location matching f.
	ListLevels(ctx context.Context, tenantID uuid.UUID, f StockFilter) (StockLevelPage, error)
	// Kardex returns one page of a variant's movements, newest first.
	Kardex(ctx context.Context, tenantID uuid.UUID, f KardexFilter) (MovementPage, error)
	MovementsByReference(ctx context.Context, tenantID uuid.UUID, ref DocumentRef) ([]*StockMovement, error)
}

// StockFilter selects the levels without a lot at a location, optionally of
// one variant.
type StockFilter struct {
	LocationID uuid.UUID
	VariantID  *uuid.UUID
	Limit      int
	Offset     int
}

// StockLevelPage is one page of levels and how many match in total.
type StockLevelPage struct {
	Items []*StockLevel
	Total int
}

// KardexFilter selects a variant's movements, optionally at one location.
type KardexFilter struct {
	VariantID  uuid.UUID
	LocationID *uuid.UUID
	Limit      int
	Offset     int
}

// MovementPage is one page of movements and how many match in total.
type MovementPage struct {
	Items []*StockMovement
	Total int
}
