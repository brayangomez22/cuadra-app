package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CountEntry is what one person counted of a variant during a stock count,
// in its base unit. Several entries of the same variant combine according to
// the count's mode; a recount leaves the earlier ones out.
type CountEntry struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	countID   uuid.UUID
	variantID uuid.UUID
	quantity  decimal.Decimal
	countedBy uuid.UUID
	countedAt time.Time
}

// NewEntry records that userID counted qty units of variantID. Zero is a
// valid count (the shelf is empty); a negative quantity, or one that does not
// fit storage, is ErrInvalidQuantity. Only an open count takes entries.
func (c *StockCount) NewEntry(id, variantID uuid.UUID, qty decimal.Decimal, userID uuid.UUID, at time.Time) (*CountEntry, error) {
	if err := c.CheckOpen(); err != nil {
		return nil, err
	}
	return RehydrateCountEntry(CountEntrySnapshot{
		ID: id, TenantID: c.tenantID, CountID: c.id, VariantID: variantID,
		Quantity: qty, CountedBy: userID, CountedAt: at,
	})
}

// CheckEntryRemoval fails unless userID may delete e while the count is
// open: whoever counted it, or someone who manages counts
// (ErrCountEntryNotYours, ErrCountNotOpen).
func (c *StockCount) CheckEntryRemoval(e *CountEntry, userID uuid.UUID, manages bool) error {
	if err := c.CheckOpen(); err != nil {
		return err
	}
	if !manages && e.countedBy != userID {
		return ErrCountEntryNotYours
	}
	return nil
}

// ID returns the entry ID (UUID v7).
func (e *CountEntry) ID() uuid.UUID { return e.id }

// TenantID returns the tenant the entry belongs to.
func (e *CountEntry) TenantID() uuid.UUID { return e.tenantID }

// CountID returns the count the entry belongs to.
func (e *CountEntry) CountID() uuid.UUID { return e.countID }

// VariantID returns the variant counted.
func (e *CountEntry) VariantID() uuid.UUID { return e.variantID }

// Quantity returns how much was counted, in the base unit.
func (e *CountEntry) Quantity() decimal.Decimal { return e.quantity }

// CountedBy returns who counted.
func (e *CountEntry) CountedBy() uuid.UUID { return e.countedBy }

// CountedAt returns when, in UTC.
func (e *CountEntry) CountedAt() time.Time { return e.countedAt }

// CountEntrySnapshot holds a stored entry's fields, for RehydrateCountEntry.
type CountEntrySnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	CountID   uuid.UUID
	VariantID uuid.UUID
	Quantity  decimal.Decimal
	CountedBy uuid.UUID
	CountedAt time.Time
}

// RehydrateCountEntry rebuilds a stored entry, checking the invariants of
// NewEntry.
func RehydrateCountEntry(s CountEntrySnapshot) (*CountEntry, error) {
	switch {
	case s.ID == uuid.Nil, s.CountID == uuid.Nil:
		return nil, ErrInvalidCountID
	case s.TenantID == uuid.Nil:
		return nil, ErrInvalidTenantID
	case s.VariantID == uuid.Nil:
		return nil, ErrInvalidVariantID
	case s.CountedBy == uuid.Nil:
		return nil, ErrInvalidUserID
	case s.Quantity.IsNegative() || !fitsStorage(s.Quantity):
		return nil, ErrInvalidQuantity
	}
	return &CountEntry{
		id: s.ID, tenantID: s.TenantID, countID: s.CountID, variantID: s.VariantID,
		quantity: s.Quantity, countedBy: s.CountedBy, countedAt: s.CountedAt.UTC(),
	}, nil
}
