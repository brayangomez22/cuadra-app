package domain

import "github.com/shopspring/decimal"

// ReturnPurchase takes out qty units of a voided purchase (purchase_void) at
// unitCost, the cost its purchase_in entered at, and undoes that entry in the
// weighted average cost: (q₀·c₀ − q·c) / (q₀ − q), rounded half-up to four
// decimals. If nothing is left, or later movements make that negative, the
// average stays. It never leaves the stock negative, whatever the policy:
// goods already sold cannot go back to the supplier. info must reference the
// purchase.
func (l *StockLevel) ReturnPurchase(qty, unitCost decimal.Decimal, info MovementInfo) (*StockMovement, error) {
	if !validQuantity(qty) {
		return nil, ErrInvalidQuantity
	}
	if !validCost(unitCost) {
		return nil, ErrInvalidUnitCost
	}
	next := l.quantity.Sub(qty)
	if err := checkBalance(next, true, StockPolicy{}); err != nil {
		return nil, err
	}
	avg := l.averageCost
	if next.IsPositive() {
		undone := l.quantity.Mul(l.averageCost).Sub(qty.Mul(unitCost)).DivRound(next, amountScale)
		if !undone.IsNegative() {
			avg = undone
		}
	}
	c, err := l.plan(MovementPurchaseVoid, qty.Neg(), unitCost, next, avg, "", info)
	if err != nil {
		return nil, err
	}
	return l.apply(c), nil
}
