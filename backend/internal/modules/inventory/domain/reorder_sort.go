package domain

import (
	"errors"
	"slices"
)

// ErrInvalidReorderSort is an order the reorder alerts do not take.
var ErrInvalidReorderSort = errors.New("inventory: invalid reorder sort")

// ReorderSort orders the reorder alerts.
type ReorderSort string

const (
	// ReorderByUrgency puts first what ran out, then the furthest below its
	// minimum in proportion.
	ReorderByUrgency ReorderSort = "urgency"
	// ReorderByToOrder puts first the most units to order to reach the
	// maximum.
	ReorderByToOrder ReorderSort = "to_order"
)

// ParseReorderSort checks s; empty is by urgency.
func ParseReorderSort(s string) (ReorderSort, error) {
	if s == "" {
		return ReorderByUrgency, nil
	}
	sort := ReorderSort(s)
	if !slices.Contains([]ReorderSort{ReorderByUrgency, ReorderByToOrder}, sort) {
		return "", ErrInvalidReorderSort
	}
	return sort, nil
}
