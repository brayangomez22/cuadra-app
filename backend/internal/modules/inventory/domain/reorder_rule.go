package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReorderRule is a variant's minimum and maximum stock at a location, in the
// product's base unit. When the stock falls to the minimum or below, the
// variant must be reordered up to the maximum.
type ReorderRule struct {
	tenantID   uuid.UUID
	variantID  uuid.UUID
	locationID uuid.UUID
	min        decimal.Decimal
	max        decimal.Decimal
}

// ReorderRuleSpec holds a rule's fields, for NewReorderRule.
type ReorderRuleSpec struct {
	TenantID   uuid.UUID
	VariantID  uuid.UUID
	LocationID uuid.UUID
	Min        decimal.Decimal
	Max        decimal.Decimal
}

// NewReorderRule creates or rehydrates a rule. It fails with
// ErrInvalidReorderRange unless 0 ≤ Min ≤ Max and both fit storage.
func NewReorderRule(s ReorderRuleSpec) (*ReorderRule, error) {
	switch {
	case s.TenantID == uuid.Nil:
		return nil, ErrInvalidTenantID
	case s.VariantID == uuid.Nil:
		return nil, ErrInvalidVariantID
	case s.LocationID == uuid.Nil:
		return nil, ErrInvalidLocationID
	case !validCost(s.Min) || !validCost(s.Max) || s.Max.LessThan(s.Min):
		return nil, ErrInvalidReorderRange
	}
	return &ReorderRule{
		tenantID:   s.TenantID,
		variantID:  s.VariantID,
		locationID: s.LocationID,
		min:        s.Min,
		max:        s.Max,
	}, nil
}

// TenantID returns the tenant the rule belongs to.
func (r *ReorderRule) TenantID() uuid.UUID { return r.tenantID }

// VariantID returns the variant the rule watches.
func (r *ReorderRule) VariantID() uuid.UUID { return r.variantID }

// LocationID returns the location whose stock the rule watches.
func (r *ReorderRule) LocationID() uuid.UUID { return r.locationID }

// Min returns the stock at or below which the variant must be reordered.
func (r *ReorderRule) Min() decimal.Decimal { return r.min }

// Max returns the stock a reorder should reach.
func (r *ReorderRule) Max() decimal.Decimal { return r.max }

// NeedsReorder reports whether onHand is at or below the minimum.
func (r *ReorderRule) NeedsReorder(onHand decimal.Decimal) bool {
	return onHand.LessThanOrEqual(r.min)
}

// Suggested returns how much to order to reach the maximum: max − onHand,
// never negative. A negative stock adds what is missing.
func (r *ReorderRule) Suggested(onHand decimal.Decimal) decimal.Decimal {
	return decimal.Max(r.max.Sub(onHand), decimal.Zero)
}

// ReorderAlert says that a variant is at or below its minimum at a location
// and how much to order.
type ReorderAlert struct {
	Rule      *ReorderRule
	OnHand    decimal.Decimal
	Suggested decimal.Decimal
}

// Alert returns the alert of onHand, and false if the variant does not need
// to be reordered.
func (r *ReorderRule) Alert(onHand decimal.Decimal) (ReorderAlert, bool) {
	if !r.NeedsReorder(onHand) {
		return ReorderAlert{}, false
	}
	return ReorderAlert{Rule: r, OnHand: onHand, Suggested: r.Suggested(onHand)}, true
}
