package domain

import "errors"

// Domain errors of the audit module. The HTTP adapter maps them to status
// codes.
var (
	// ErrUnknownAction is an action outside the catalog.
	ErrUnknownAction = errors.New("audit: action outside the catalog")
	// ErrInvalidDetails is details without a key the action requires, with
	// a key it does not declare or with an empty or too long value.
	ErrInvalidDetails = errors.New("audit: details do not match the action")
	// ErrInvalidEvent is an event without tenant, user, role, time or the
	// id of the entity its action is about.
	ErrInvalidEvent = errors.New("audit: invalid event")
	// ErrInvalidFilter is a query with an inverted date range.
	ErrInvalidFilter = errors.New("audit: invalid event filter")
)
