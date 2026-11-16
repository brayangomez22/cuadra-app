package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EventFilter narrows a tenant's events: each set field must match.
// OccurredFrom is inclusive and OccurredBefore exclusive.
type EventFilter struct {
	TenantID       uuid.UUID
	Action         *Action
	UserID         *uuid.UUID
	EntityID       *uuid.UUID
	OccurredFrom   *time.Time
	OccurredBefore *time.Time
	Limit, Offset  int
}

// EventPage is a page of events, newest first, and how many match.
type EventPage struct {
	Items []*Event
	Total int
}

// EventRepository stores the audit log. It only appends: there is no way
// to change or delete an event (the database refuses it too).
type EventRepository interface {
	// Append stores e. With a ctx that carries a transaction (db.InTenantTx)
	// it joins it, so the event commits or rolls back with the operation.
	Append(ctx context.Context, e *Event) error
	List(ctx context.Context, f EventFilter) (EventPage, error)
}
