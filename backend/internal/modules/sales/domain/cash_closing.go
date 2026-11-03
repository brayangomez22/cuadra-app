package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CloseCashInput is a closing: who closes, the count of the drawer, the
// session's activity (which the repository reads while the session is
// locked) and optional notes. ManagesCash is whether who closes may close
// another user's session (one left open), from their role.
type CloseCashInput struct {
	UserID      uuid.UUID
	ManagesCash bool
	Count       CashCount
	Activity    CashActivity
	Notes       string
	At          time.Time
}

// CashClosing is the closing of a session. Expected is ExpectedCash of the
// activity, Counted the count's total and Difference = Counted − Expected:
// negative is a shortage, positive an overage.
type CashClosing struct {
	ClosedBy   uuid.UUID
	ClosedAt   time.Time
	Notes      string
	Count      CashCount
	Activity   CashActivity
	Expected   decimal.Decimal
	Counted    decimal.Decimal
	Difference decimal.Decimal
}

// Result says whether the count matched the expected cash, fell short or was
// over.
func (c CashClosing) Result() CashResult {
	switch c.Difference.Sign() {
	case -1:
		return CashShortage
	case 1:
		return CashOverage
	}
	return CashExact
}

// SalesTotal is what the session's confirmed sales add up to: what every
// payment method collected, cash net of the change and credit included.
func (a CashActivity) SalesTotal() decimal.Decimal {
	total := decimal.Zero
	for _, m := range a.ByMethod {
		total = total.Add(m.Amount)
	}
	return total
}
