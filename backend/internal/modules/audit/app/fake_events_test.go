package app_test

import (
	"context"
	"slices"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
)

// fakeEvents is an in-memory audit log of every tenant.
type fakeEvents struct {
	stored []*domain.Event
	// fail makes Append fail with it.
	fail error
}

func (f *fakeEvents) Append(_ context.Context, e *domain.Event) error {
	if f.fail != nil {
		return f.fail
	}
	f.stored = append(f.stored, e)
	return nil
}

func (f *fakeEvents) List(_ context.Context, filter domain.EventFilter) (domain.EventPage, error) {
	var match []*domain.Event
	for _, e := range f.stored {
		if matches(e, filter) {
			match = append(match, e)
		}
	}
	slices.Reverse(match)
	page := domain.EventPage{Total: len(match)}
	end := min(filter.Offset+filter.Limit, len(match))
	if filter.Offset < end {
		page.Items = match[filter.Offset:end]
	}
	return page, nil
}

func matches(e *domain.Event, f domain.EventFilter) bool {
	switch {
	case e.TenantID() != f.TenantID,
		f.Action != nil && e.Action() != *f.Action,
		f.UserID != nil && e.UserID() != *f.UserID,
		f.EntityID != nil && (e.EntityID() == nil || *e.EntityID() != *f.EntityID),
		f.OccurredFrom != nil && e.OccurredAt().Before(*f.OccurredFrom),
		f.OccurredBefore != nil && !e.OccurredAt().Before(*f.OccurredBefore):
		return false
	}
	return true
}
