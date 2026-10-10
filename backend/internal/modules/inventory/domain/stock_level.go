package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// StockLevel is a product's stock at a location and its weighted average
// cost. Every change goes through Receive, Issue, Adjust or Transfer, which
// return the StockMovement that records it, so the level and the kardex
// cannot disagree. On error the level is unchanged.
type StockLevel struct {
	tenantID    uuid.UUID
	productID   uuid.UUID
	locationID  uuid.UUID
	quantity    decimal.Decimal
	averageCost decimal.Decimal
}

// NewStockLevel creates an empty level for a product at a location.
func NewStockLevel(tenantID, productID, locationID uuid.UUID) (*StockLevel, error) {
	return newStockLevel(StockLevelSnapshot{TenantID: tenantID, ProductID: productID, LocationID: locationID})
}

// newStockLevel checks the invariants shared by NewStockLevel and
// RehydrateStockLevel. A negative quantity is valid: the tenant may allow it,
// or may have allowed it before changing its policy.
func newStockLevel(s StockLevelSnapshot) (*StockLevel, error) {
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if s.ProductID == uuid.Nil {
		return nil, ErrInvalidProductID
	}
	if s.LocationID == uuid.Nil {
		return nil, ErrInvalidLocationID
	}
	if !fitsStorage(s.Quantity) || !validCost(s.AverageCost) {
		return nil, ErrInvalidStockLevel
	}
	return &StockLevel{
		tenantID:    s.TenantID,
		productID:   s.ProductID,
		locationID:  s.LocationID,
		quantity:    s.Quantity,
		averageCost: s.AverageCost,
	}, nil
}

// TenantID returns the tenant the level belongs to.
func (l *StockLevel) TenantID() uuid.UUID { return l.tenantID }

// ProductID returns the product whose stock this is.
func (l *StockLevel) ProductID() uuid.UUID { return l.productID }

// LocationID returns the location the stock is at.
func (l *StockLevel) LocationID() uuid.UUID { return l.locationID }

// Quantity returns the stock on hand, in the product's base unit. It is
// negative only if the tenant's policy allowed it.
func (l *StockLevel) Quantity() decimal.Decimal { return l.quantity }

// AverageCost returns the weighted average unit cost, without IVA.
func (l *StockLevel) AverageCost() decimal.Decimal { return l.averageCost }

// Receive registers an entry of qty units bought at unitCost (purchase_in),
// including the initial stock, and updates the weighted average cost.
func (l *StockLevel) Receive(qty, unitCost decimal.Decimal, info MovementInfo) (*StockMovement, error) {
	c, err := l.planIn(MovementPurchaseIn, qty, unitCost, info)
	if err != nil {
		return nil, err
	}
	return l.apply(c), nil
}

// Issue registers a sale of qty units (sale_out), valued at the average cost.
// It fails with ErrInsufficientStock if the stock would go negative and the
// policy does not allow it.
func (l *StockLevel) Issue(qty decimal.Decimal, policy StockPolicy, info MovementInfo) (*StockMovement, error) {
	c, err := l.planOut(MovementSaleOut, qty, policy, info)
	if err != nil {
		return nil, err
	}
	return l.apply(c), nil
}

// Adjust corrects the stock by delta (positive or negative) for the given
// reason, which is required. It is valued at the average cost and does not
// change it; a negative delta follows the policy like Issue.
func (l *StockLevel) Adjust(delta decimal.Decimal, reason string, policy StockPolicy, info MovementInfo) (*StockMovement, error) {
	reason, err := normalizeReason(reason)
	if err != nil {
		return nil, err
	}
	if delta.IsZero() || !fitsStorage(delta) {
		return nil, ErrInvalidQuantity
	}
	next := l.quantity.Add(delta)
	if err := checkBalance(next, delta.IsNegative(), policy); err != nil {
		return nil, err
	}
	c, err := l.plan(MovementAdjustment, delta, l.averageCost, next, l.averageCost, reason, info)
	if err != nil {
		return nil, err
	}
	return l.apply(c), nil
}

// Transfer moves qty units of a product from one location to another. It
// creates a transfer_out at the origin and a transfer_in at the destination,
// both valued at the origin's average cost and sharing a new transfer
// reference. The origin follows the policy like Issue. Both levels change or
// neither does.
func Transfer(from, to *StockLevel, qty decimal.Decimal, policy StockPolicy, userID uuid.UUID, occurredAt time.Time) (out, in *StockMovement, err error) {
	if from.tenantID != to.tenantID || from.productID != to.productID {
		return nil, nil, ErrTransferMismatch
	}
	if from.locationID == to.locationID {
		return nil, nil, ErrSameLocationTransfer
	}
	transferID, err := uuid.NewV7()
	if err != nil {
		return nil, nil, err
	}
	info := MovementInfo{
		UserID:     userID,
		Reference:  &DocumentRef{Kind: DocumentTransfer, ID: transferID},
		OccurredAt: occurredAt,
	}
	outChange, err := from.planOut(MovementTransferOut, qty, policy, info)
	if err != nil {
		return nil, nil, err
	}
	inChange, err := to.planIn(MovementTransferIn, qty, from.averageCost, info)
	if err != nil {
		return nil, nil, err
	}
	return from.apply(outChange), to.apply(inChange), nil
}

// change is a validated movement and the level's state after it, computed
// before mutating anything so a failed operation leaves the level unchanged.
type change struct {
	movement    *StockMovement
	quantity    decimal.Decimal
	averageCost decimal.Decimal
}

func (l *StockLevel) apply(c change) *StockMovement {
	l.quantity, l.averageCost = c.quantity, c.averageCost
	return c.movement
}

// planIn computes an entry of qty units at unitCost. The new average cost is
// (q₀·c₀ + q·c) / (q₀ + q), rounded half-up to four decimals; if the stock
// was zero or negative, it is the entry's cost.
func (l *StockLevel) planIn(t MovementType, qty, unitCost decimal.Decimal, info MovementInfo) (change, error) {
	if !validQuantity(qty) {
		return change{}, ErrInvalidQuantity
	}
	if !validCost(unitCost) {
		return change{}, ErrInvalidUnitCost
	}
	next := l.quantity.Add(qty)
	if !fitsStorage(next) {
		return change{}, ErrInvalidQuantity
	}
	avg := unitCost
	if l.quantity.IsPositive() {
		avg = l.quantity.Mul(l.averageCost).Add(qty.Mul(unitCost)).DivRound(next, amountScale)
	}
	return l.plan(t, qty, unitCost, next, avg, "", info)
}

// planOut computes an exit of qty units at the average cost.
func (l *StockLevel) planOut(t MovementType, qty decimal.Decimal, policy StockPolicy, info MovementInfo) (change, error) {
	if !validQuantity(qty) {
		return change{}, ErrInvalidQuantity
	}
	next := l.quantity.Sub(qty)
	if err := checkBalance(next, true, policy); err != nil {
		return change{}, err
	}
	return l.plan(t, qty.Neg(), l.averageCost, next, l.averageCost, "", info)
}

// checkBalance validates the stock left after a movement: it must fit
// storage and, for an exit, not go negative unless the policy allows it.
func checkBalance(next decimal.Decimal, isExit bool, policy StockPolicy) error {
	if !fitsStorage(next) {
		return ErrInvalidQuantity
	}
	if isExit && next.IsNegative() && !policy.AllowNegative {
		return ErrInsufficientStock
	}
	return nil
}

// plan builds the movement for a change; newMovement validates the rest
// (user, reference, signs).
func (l *StockLevel) plan(t MovementType, qty, unitCost, next, avg decimal.Decimal, reason string, info MovementInfo) (change, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return change{}, err
	}
	m, err := newMovement(StockMovementSnapshot{
		ID:               id,
		TenantID:         l.tenantID,
		ProductID:        l.productID,
		LocationID:       l.locationID,
		Type:             t,
		Quantity:         qty,
		UnitCost:         unitCost,
		BalanceAfter:     next,
		AverageCostAfter: avg,
		Reason:           reason,
		Reference:        info.Reference,
		UserID:           info.UserID,
		OccurredAt:       info.OccurredAt,
	})
	if err != nil {
		return change{}, err
	}
	return change{movement: m, quantity: next, averageCost: avg}, nil
}

// StockLevelSnapshot holds a stored level's fields, for RehydrateStockLevel.
type StockLevelSnapshot struct {
	TenantID    uuid.UUID
	ProductID   uuid.UUID
	LocationID  uuid.UUID
	Quantity    decimal.Decimal
	AverageCost decimal.Decimal
}

// RehydrateStockLevel rebuilds a stored level, checking the same invariants
// as NewStockLevel.
func RehydrateStockLevel(s StockLevelSnapshot) (*StockLevel, error) { return newStockLevel(s) }
