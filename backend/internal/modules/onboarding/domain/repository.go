package domain

import (
	"context"

	"github.com/google/uuid"
)

// TenantState is what the checklist reads of a tenant: its business
// template and the facts of its data. The invoicing facts come apart, from
// the invoicing module.
type TenantState struct {
	Template string
	Facts    Facts
}

// StateReader reads a tenant's state from the other modules' tables.
type StateReader interface {
	TenantState(ctx context.Context, tenantID uuid.UUID) (TenantState, error)
}

// DismissalRepository keeps who hid their checklist.
type DismissalRepository interface {
	// Get returns the user's dismissal; ok is false if they never hid it.
	Get(ctx context.Context, tenantID, userID uuid.UUID) (d Dismissal, ok bool, err error)
	// Save stores d, replacing the user's earlier one.
	Save(ctx context.Context, d Dismissal) error
}
