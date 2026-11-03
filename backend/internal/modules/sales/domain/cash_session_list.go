package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CashSessionFilter is a page of a tenant's sessions, the latest opened
// first. A nil field does not filter; Status "" means open and closed.
// OpenedFrom is included and OpenedBefore excluded.
type CashSessionFilter struct {
	TenantID     uuid.UUID
	LocationID   *uuid.UUID
	Status       CashSessionStatus
	OpenedFrom   *time.Time
	OpenedBefore *time.Time
	Limit        int
	Offset       int
}

// CashSessionSummary is a session as the history lists it. The closing's
// fields are set only once it is closed; SalesTotal is the closing's
// CashActivity.SalesTotal.
type CashSessionSummary struct {
	ID           uuid.UUID
	LocationID   uuid.UUID
	UserID       uuid.UUID
	Status       CashSessionStatus
	OpeningFloat decimal.Decimal
	OpenedAt     time.Time
	ClosedAt     *time.Time
	ClosedBy     *uuid.UUID
	SalesCount   int
	SalesTotal   decimal.Decimal
	Difference   *decimal.Decimal
}

// Result says how the count of a closed session came out; ok is false while
// it is open.
func (s CashSessionSummary) Result() (result CashResult, ok bool) {
	if s.Difference == nil {
		return "", false
	}
	return CashClosing{Difference: *s.Difference}.Result(), true
}

// CashSummaryOf is the summary of a session.
func CashSummaryOf(s *CashSession) CashSessionSummary {
	out := CashSessionSummary{
		ID: s.ID(), LocationID: s.LocationID(), UserID: s.UserID(), Status: s.Status(),
		OpeningFloat: s.OpeningFloat(), OpenedAt: s.OpenedAt(),
	}
	if c := s.Closing(); c != nil {
		out.ClosedAt, out.ClosedBy, out.Difference = &c.ClosedAt, &c.ClosedBy, &c.Difference
		out.SalesCount, out.SalesTotal = c.Activity.Sales, c.Activity.SalesTotal()
	}
	return out
}
