package domain

import (
	"bytes"
	"slices"
	"time"

	"github.com/google/uuid"
)

// CountStatus is where a stock count is in its life: open while people
// count, then approved (its adjustments are in the kardex) or canceled.
type CountStatus string

// Stock count statuses.
const (
	CountOpen     CountStatus = "open"
	CountApproved CountStatus = "approved"
	CountCanceled CountStatus = "canceled"
)

// ParseCountStatus returns the status with code s. Matching is exact.
func ParseCountStatus(s string) (CountStatus, error) {
	switch st := CountStatus(s); st {
	case CountOpen, CountApproved, CountCanceled:
		return st, nil
	}
	return "", ErrInvalidCountStatus
}

// CountMode says what two counts of the same variant by different people
// mean: in sum mode each counted a part (two shelves) and they add up; in
// review mode both counted everything, and if they disagree the variant must
// be recounted.
type CountMode string

// Stock count modes.
const (
	CountSum    CountMode = "sum"
	CountReview CountMode = "review"
)

// ParseCountMode returns the mode with code s. Matching is exact.
func ParseCountMode(s string) (CountMode, error) {
	switch m := CountMode(s); m {
	case CountSum, CountReview:
		return m, nil
	}
	return "", ErrInvalidCountMode
}

// StockCount is a physical count of the stock at a location: of every
// variant there (full) or of the variants of some categories (partial).
// Counters record what they count (CountEntry) without seeing the expected
// stock; an owner or admin reviews the differences and approves the count,
// which adjusts the stock, or cancels it.
type StockCount struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	locationID  uuid.UUID
	categoryIDs []uuid.UUID
	mode        CountMode
	status      CountStatus
	startedBy   uuid.UUID
	startedAt   time.Time
	closedBy    uuid.UUID
	closedAt    time.Time
	summary     *CountSummary
}

// StockCountSpec holds the fields of a new count. CategoryIDs is empty for a
// full count.
type StockCountSpec struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	LocationID  uuid.UUID
	CategoryIDs []uuid.UUID
	Mode        CountMode
	StartedBy   uuid.UUID
	StartedAt   time.Time
}

// NewStockCount starts an open count. Repeated categories are kept once.
func NewStockCount(s StockCountSpec) (*StockCount, error) {
	if err := s.check(); err != nil {
		return nil, err
	}
	categories := slices.Clone(s.CategoryIDs)
	slices.SortFunc(categories, compareIDs)
	return &StockCount{
		id:          s.ID,
		tenantID:    s.TenantID,
		locationID:  s.LocationID,
		categoryIDs: slices.Compact(categories),
		mode:        s.Mode,
		status:      CountOpen,
		startedBy:   s.StartedBy,
		startedAt:   s.StartedAt.UTC(),
	}, nil
}

func (s StockCountSpec) check() error {
	switch {
	case s.ID == uuid.Nil:
		return ErrInvalidCountID
	case s.TenantID == uuid.Nil:
		return ErrInvalidTenantID
	case s.LocationID == uuid.Nil:
		return ErrInvalidLocationID
	case s.StartedBy == uuid.Nil:
		return ErrInvalidUserID
	case slices.Contains(s.CategoryIDs, uuid.Nil):
		return ErrInvalidCategoryID
	}
	_, err := ParseCountMode(string(s.Mode))
	return err
}

// ID returns the count ID (UUID v7).
func (c *StockCount) ID() uuid.UUID { return c.id }

// TenantID returns the tenant the count belongs to.
func (c *StockCount) TenantID() uuid.UUID { return c.tenantID }

// LocationID returns the location whose stock is counted.
func (c *StockCount) LocationID() uuid.UUID { return c.locationID }

// CategoryIDs returns a copy of the categories of a partial count, sorted;
// empty (not nil) for a full count.
func (c *StockCount) CategoryIDs() []uuid.UUID { return append([]uuid.UUID{}, c.categoryIDs...) }

// IsPartial reports whether the count covers only some categories.
func (c *StockCount) IsPartial() bool { return len(c.categoryIDs) > 0 }

// Mode returns how counts of the same variant by different people combine.
func (c *StockCount) Mode() CountMode { return c.mode }

// Status returns where the count is in its life.
func (c *StockCount) Status() CountStatus { return c.status }

// StartedBy returns who started the count.
func (c *StockCount) StartedBy() uuid.UUID { return c.startedBy }

// StartedAt returns when the snapshot of the stock was taken, in UTC.
func (c *StockCount) StartedAt() time.Time { return c.startedAt }

// ClosedBy returns who approved or canceled the count; uuid.Nil while open.
func (c *StockCount) ClosedBy() uuid.UUID { return c.closedBy }

// ClosedAt returns when the count was approved or canceled, in UTC; the
// zero time while open.
func (c *StockCount) ClosedAt() time.Time { return c.closedAt }

// Summary returns the accuracy of an approved count; false otherwise.
func (c *StockCount) Summary() (CountSummary, bool) {
	if c.summary == nil {
		return CountSummary{}, false
	}
	return *c.summary, true
}

// CheckOpen fails with ErrCountNotOpen unless the count is open.
func (c *StockCount) CheckOpen() error {
	if c.status != CountOpen {
		return ErrCountNotOpen
	}
	return nil
}

// StockCountSnapshot holds a stored count's fields, for RehydrateStockCount.
type StockCountSnapshot struct {
	StockCountSpec
	Status   CountStatus
	ClosedBy uuid.UUID
	ClosedAt time.Time
	Summary  *CountSummary
}

// RehydrateStockCount rebuilds a stored count, checking the invariants of
// NewStockCount and of its status: a closed count has who and when, and only
// an approved one has a summary.
func RehydrateStockCount(s StockCountSnapshot) (*StockCount, error) {
	c, err := NewStockCount(s.StockCountSpec)
	if err != nil {
		return nil, err
	}
	if _, err := ParseCountStatus(string(s.Status)); err != nil {
		return nil, err
	}
	closed := s.Status != CountOpen
	if closed == (s.ClosedBy == uuid.Nil) || closed == s.ClosedAt.IsZero() ||
		(s.Status == CountApproved) != (s.Summary != nil) {
		return nil, ErrInvalidStockCount
	}
	c.status, c.closedBy = s.Status, s.ClosedBy
	if closed {
		c.closedAt = s.ClosedAt.UTC()
	}
	if s.Summary != nil {
		summary := *s.Summary
		c.summary = &summary
	}
	return c, nil
}

// compareIDs orders ids by their bytes, the order Postgres sorts uuids in.
func compareIDs(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) }
