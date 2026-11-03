package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ErrInvalidCashStatus reports a history filtered by an unknown status.
var ErrInvalidCashStatus = errors.New("sales: invalid cash session status filter")

// UserDirectory names the tenant's users, which sales keeps by id. Names
// returns the name of each user of tenantID among ids; a missing one is not
// the tenant's.
type UserDirectory interface {
	UserNames(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// CashSessionQuery is a page of the history. Status is "open", "closed" or
// "" for both. From and To are days in Colombia (their year, month and day),
// both included, on the day the session opened. The HTTP adapter bounds
// Limit and Offset.
type CashSessionQuery struct {
	LocationID *uuid.UUID
	Status     string
	From, To   *time.Time
	Limit      int
	Offset     int
}

// CashSessionRow is a session of the history with its cashier's name ("" if
// the user no longer exists).
type CashSessionRow struct {
	domain.CashSessionSummary
	CashierName string
}

// CashSessionPage is a page of the history and how many sessions match.
type CashSessionPage struct {
	Items []CashSessionRow
	Total int
}

// ListCashSessions lists the tenant's cash sessions, the latest opened
// first, each with its cashier's name and, once closed, its sales and how the
// count came out.
func (s *Service) ListCashSessions(ctx context.Context, a Actor, q CashSessionQuery) (_ CashSessionPage, err error) {
	ctx, span := s.startSpan(ctx, "ListCashSessions", a)
	defer func() { usecase.End(span, err) }()

	status := domain.CashSessionStatus(q.Status)
	if status != "" && status != domain.CashSessionOpen && status != domain.CashSessionClosed {
		return CashSessionPage{}, ErrInvalidCashStatus
	}
	items, total, err := s.CashSessions.Search(ctx, domain.CashSessionFilter{
		TenantID: a.TenantID, LocationID: q.LocationID, Status: status,
		OpenedFrom: dayStart(q.From, 0), OpenedBefore: dayStart(q.To, 1), Limit: q.Limit, Offset: q.Offset,
	})
	if err != nil {
		return CashSessionPage{}, err
	}
	names, err := s.Users.UserNames(ctx, a.TenantID, cashiers(items))
	if err != nil {
		return CashSessionPage{}, err
	}
	page := CashSessionPage{Items: make([]CashSessionRow, len(items)), Total: total}
	for i, item := range items {
		page.Items[i] = CashSessionRow{CashSessionSummary: item, CashierName: names[item.UserID]}
	}
	return page, nil
}

// dayStart is when day (in Colombia) plus days starts, in UTC; nil for nil.
func dayStart(day *time.Time, days int) *time.Time {
	if day == nil {
		return nil
	}
	y, m, d := day.Date()
	start := time.Date(y, m, d+days, 0, 0, 0, 0, bogota).UTC()
	return &start
}

func cashiers(items []domain.CashSessionSummary) []uuid.UUID {
	seen := map[uuid.UUID]bool{}
	var out []uuid.UUID
	for _, item := range items {
		if !seen[item.UserID] {
			seen[item.UserID] = true
			out = append(out, item.UserID)
		}
	}
	return out
}
