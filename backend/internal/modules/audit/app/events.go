package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// bogota is Colombia's time zone (no daylight saving), where the days of a
// query start.
var bogota = time.FixedZone("America/Bogota", -5*3600)

// EventQuery narrows ListEvents. Action is a name of the catalog or "";
// From and To are days in Colombia (their year, month and day), both
// included. Limit and Offset come validated by the API.
type EventQuery struct {
	Action           string
	UserID, EntityID *uuid.UUID
	From, To         *time.Time
	Limit, Offset    int
}

// ListEvents returns a page of the actor's tenant's events, newest first.
// The permission is checked by the API (audit:read).
func (s *Service) ListEvents(ctx context.Context, a Actor, q EventQuery) (_ domain.EventPage, err error) {
	ctx, span := s.startSpan(ctx, "ListEvents", a)
	defer func() { usecase.End(span, err) }()

	f, err := q.filter(a.TenantID)
	if err != nil {
		return domain.EventPage{}, err
	}
	return s.Events.List(ctx, f)
}

func (q EventQuery) filter(tenantID uuid.UUID) (domain.EventFilter, error) {
	f := domain.EventFilter{
		TenantID: tenantID, UserID: q.UserID, EntityID: q.EntityID,
		OccurredFrom: dayStart(q.From, 0), OccurredBefore: dayStart(q.To, 1),
		Limit: q.Limit, Offset: q.Offset,
	}
	if q.Action != "" {
		action, err := domain.ParseAction(q.Action)
		if err != nil {
			return domain.EventFilter{}, err
		}
		f.Action = &action
	}
	if f.OccurredFrom != nil && f.OccurredBefore != nil && !f.OccurredFrom.Before(*f.OccurredBefore) {
		return domain.EventFilter{}, domain.ErrInvalidFilter
	}
	return f, nil
}

// dayStart is the instant the day days after day starts in Colombia; nil
// for a nil day.
func dayStart(day *time.Time, days int) *time.Time {
	if day == nil {
		return nil
	}
	y, m, d := day.Date()
	start := time.Date(y, m, d+days, 0, 0, 0, 0, bogota).UTC()
	return &start
}
