package domain

import (
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
)

// EventInput is what NewEvent validates. UserID and Role are who did it,
// from the access token; EntityID is nil only for an action whose entity
// has no uuid; LocationID is where, when the operation has a location.
type EventInput struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	UserID     uuid.UUID
	Role       string
	Action     Action
	EntityID   *uuid.UUID
	LocationID *uuid.UUID
	OccurredAt time.Time
	Details    map[string]string
}

// Event is an entry of a tenant's audit log. It is immutable.
type Event struct {
	in EventInput
}

// NewEvent validates in against the catalog: a known action, its entity's
// id when it has one and exactly the details the action declares.
func NewEvent(in EventInput) (*Event, error) {
	spec, ok := catalog[in.Action]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownAction, in.Action)
	}
	if in.ID == uuid.Nil || in.TenantID == uuid.Nil || in.UserID == uuid.Nil ||
		in.Role == "" || in.OccurredAt.IsZero() {
		return nil, ErrInvalidEvent
	}
	if spec.hasEntityID != (in.EntityID != nil) {
		return nil, fmt.Errorf("%w: entity id of %q", ErrInvalidEvent, in.Action)
	}
	if err := spec.checkDetails(in.Details); err != nil {
		return nil, err
	}
	in.OccurredAt = in.OccurredAt.UTC()
	in.Details = maps.Clone(in.Details)
	in.EntityID = clone(in.EntityID)
	in.LocationID = clone(in.LocationID)
	return &Event{in: in}, nil
}

func clone(id *uuid.UUID) *uuid.UUID {
	if id == nil {
		return nil
	}
	c := *id
	return &c
}

// ID returns the event's id.
func (e *Event) ID() uuid.UUID { return e.in.ID }

// TenantID returns the tenant whose log holds the event.
func (e *Event) TenantID() uuid.UUID { return e.in.TenantID }

// UserID returns who did the operation.
func (e *Event) UserID() uuid.UUID { return e.in.UserID }

// Role returns the role they had when they did it.
func (e *Event) Role() string { return e.in.Role }

// Action returns what they did.
func (e *Event) Action() Action { return e.in.Action }

// EntityType returns the kind of entity the action is about.
func (e *Event) EntityType() string { return e.in.Action.EntityType() }

// EntityID returns the entity, nil for an action about a module.
func (e *Event) EntityID() *uuid.UUID { return clone(e.in.EntityID) }

// LocationID returns where, nil when the operation has no location.
func (e *Event) LocationID() *uuid.UUID { return clone(e.in.LocationID) }

// OccurredAt returns when, in UTC.
func (e *Event) OccurredAt() time.Time { return e.in.OccurredAt }

// Details returns the figures of the event, by the action's keys.
func (e *Event) Details() map[string]string { return maps.Clone(e.in.Details) }

// Snapshot returns the event's fields, for the repository to store.
func (e *Event) Snapshot() EventInput {
	s := e.in
	s.Details = maps.Clone(s.Details)
	return s
}

// RestoreEvent rebuilds an event the repository read: it was valid when
// stored, so it is not validated again (the catalog only grows).
func RestoreEvent(s EventInput) *Event { return &Event{in: s} }
