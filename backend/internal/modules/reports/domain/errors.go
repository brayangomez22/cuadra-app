package domain

import "errors"

// The reports' errors.
var (
	// ErrInvalidPeriod is a period whose last day is before its first, or
	// longer than MaxPeriodDays.
	ErrInvalidPeriod = errors.New("reports: invalid period")
	// ErrInvalidFilter is a filter a report does not take, or a missing one
	// it requires.
	ErrInvalidFilter = errors.New("reports: invalid filter")
	// ErrCostsHidden is a report, or an order, that needs the costs the
	// caller may not see.
	ErrCostsHidden = errors.New("reports: costs hidden")
)
