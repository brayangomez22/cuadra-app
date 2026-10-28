package domain

import (
	"context"

	"github.com/google/uuid"
)

// StockCountRepository persists stock counts and their lines. A count of
// another tenant is reported as ErrCountNotFound.
type StockCountRepository interface {
	// Start stores the open count c with its snapshot: one line per variant
	// with a level (without a lot) at the location in a full count, or per
	// variant of variantIDs in a partial one. The start of each line is its
	// stock at c.StartedAt according to the kardex (zero without movements)
	// and its current average cost. It returns how many lines it took;
	// ErrCountInProgress if the location already has an open count,
	// ErrLocationNotFound if the location does not exist in the tenant.
	Start(ctx context.Context, c *StockCount, variantIDs []uuid.UUID) (int, error)
	// Get returns ErrCountNotFound when the count does not exist in tenantID.
	Get(ctx context.Context, tenantID, id uuid.UUID) (*StockCount, error)
	// List returns one page of the counts matching f, newest first.
	List(ctx context.Context, tenantID uuid.UUID, f CountFilter) (StockCountPage, error)
	// Progress returns how many lines the count has and how many have an
	// entry in their current round.
	Progress(ctx context.Context, tenantID, id uuid.UUID) (CountProgress, error)
	// Sheet returns every line of c, by variant, with the totals of each
	// counter in the line's current round and the net quantity of the kardex
	// at the location after c.StartedAt and up to the line's last entry.
	Sheet(ctx context.Context, c *StockCount) ([]SheetLine, error)
	// Recount starts a new round of the lines of variantIDs, which leaves
	// their entries so far out; ErrVariantNotInCount if one has no line.
	Recount(ctx context.Context, tenantID, id uuid.UUID, variantIDs []uuid.UUID) error
	// Close runs fn with the count locked against new entries and stores the
	// state fn leaves it in (approved or canceled), all in one transaction:
	// the ctx fn receives joins it, so the stock changes fn makes commit or
	// roll back with the count. ErrCountNotFound if it does not exist.
	Close(ctx context.Context, tenantID, id uuid.UUID, fn func(context.Context, *StockCount) error) error
}

// CountEntryRepository persists what counters count.
type CountEntryRepository interface {
	// Add stores e in the current round of its variant's line. If the
	// variant has no line, with addLine it first adds one starting at zero
	// (a full count), and otherwise fails with ErrVariantNotInCount. It
	// fails with ErrCountNotOpen if the count was closed meanwhile, and with
	// ErrVariantNotFound if the variant does not exist in the tenant.
	Add(ctx context.Context, e *CountEntry, addLine bool) error
	// Get returns ErrCountEntryNotFound when the entry does not exist in the
	// count.
	Get(ctx context.Context, tenantID, countID, id uuid.UUID) (*CountEntry, error)
	// Delete removes the entry; ErrCountEntryNotFound if it is gone.
	Delete(ctx context.Context, e *CountEntry) error
	// List returns one page of the entries of the current rounds matching f,
	// newest first.
	List(ctx context.Context, tenantID uuid.UUID, f EntryFilter) (CountEntryPage, error)
}

// CountFilter selects counts, optionally of one location and one status.
type CountFilter struct {
	LocationID *uuid.UUID
	Status     *CountStatus
	Limit      int
	Offset     int
}

// StockCountPage is one page of counts and how many match in total.
type StockCountPage struct {
	Items []*StockCount
	Total int
}

// CountProgress is how far a count went: its lines and how many were
// counted.
type CountProgress struct {
	Lines   int
	Counted int
}

// EntryFilter selects the entries of a count, optionally of one variant or
// one counter.
type EntryFilter struct {
	CountID   uuid.UUID
	VariantID *uuid.UUID
	CountedBy *uuid.UUID
	Limit     int
	Offset    int
}

// CountEntryPage is one page of entries and how many match in total.
type CountEntryPage struct {
	Items []*CountEntry
	Total int
}
