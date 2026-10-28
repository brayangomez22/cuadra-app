package app

import (
	"context"
	"errors"
	"iter"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// The imports module reaches the catalog, the inventory and the activable
// modules through these ports; internal/server satisfies them, so no
// module imports another.

// ErrLocationNotFound is the Stock port's error for a location that does
// not exist, or is inactive, in the tenant.
var ErrLocationNotFound = errors.New("imports: location not found")

// Catalog checks and writes the products of an import. Both take the
// groups of one batch and answer one result per group, in order.
type Catalog interface {
	// Layout returns the tenant's attributes, variant axes, units and
	// default IVA; the Service fills in Variants.
	Layout(ctx context.Context, a Actor) (domain.Layout, error)
	// CheckProducts says what writing each group would do, without
	// writing.
	CheckProducts(ctx context.Context, a Actor, groups []domain.Group, p domain.ExistingPolicy) ([]ProductResult, error)
	// ImportProducts writes the groups in order and stops at the first one
	// with a Problem, which is the last result. With a ctx from
	// Transactor.InTx it joins the batch's transaction.
	ImportProducts(ctx context.Context, a Actor, groups []domain.Group, p domain.ExistingPolicy) ([]ProductResult, error)
}

// Outcome is what writing a group does to the catalog.
type Outcome string

// The outcomes.
const (
	OutcomeCreated Outcome = "created"
	OutcomeUpdated Outcome = "updated"
	OutcomeSkipped Outcome = "skipped"
)

// ProductResult is the result of one group: its outcome and its variants,
// in the group's order, or the Problem the catalog found with it.
type ProductResult struct {
	Outcome  Outcome
	Variants []VariantResult
	Problem  *Problem
}

// VariantResult is a variant of a group in the catalog: Created is false
// for one that already existed (it gets no initial stock).
type VariantResult struct {
	ID      uuid.UUID
	Created bool
}

// Problem is why the catalog rejects a group, as the catalog describes it
// to the user.
type Problem struct {
	Code    string
	Message string
}

// Stock checks the import's location and receives the initial stock.
type Stock interface {
	// RequireLocation fails with ErrLocationNotFound unless id is an
	// active location of the tenant.
	RequireLocation(ctx context.Context, a Actor, id uuid.UUID) error
	// ReceiveInitial registers the entries at locationID, joining the
	// batch's transaction like Catalog.ImportProducts.
	ReceiveInitial(ctx context.Context, a Actor, locationID uuid.UUID, entries []StockEntry) error
}

// StockEntry is the initial stock of a new variant, in its base unit, at
// the variant's cost.
type StockEntry struct {
	VariantID uuid.UUID
	Quantity  decimal.Decimal
	UnitCost  decimal.Decimal
}

// Modules reports which activable modules a tenant has on.
type Modules interface {
	IsActive(ctx context.Context, tenantID uuid.UUID, id string) (bool, error)
}

// Sheets reads and writes the import's files.
type Sheets interface {
	// Rows reads the rows of data in format, header first;
	// domain.ErrUnreadableFile if it is not such a file.
	Rows(format domain.Format, data []byte) (Sheet, error)
	// Template writes an empty file in format with l's columns and a help
	// of their values.
	Template(format domain.Format, l domain.Layout) ([]byte, error)
}

// Sheet is a file being read: its rows, one at a time, and how it writes
// numbers.
type Sheet struct {
	Rows  iter.Seq2[[]string, error]
	Style domain.NumberStyle
}

// Transactor runs fn in one transaction of tenantID; every repository and
// port called with fn's ctx joins it.
type Transactor interface {
	InTx(ctx context.Context, tenantID uuid.UUID, fn func(context.Context) error) error
}
