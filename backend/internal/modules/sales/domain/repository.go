package domain

import (
	"context"

	"github.com/google/uuid"
)

// SaleRepository persists sales. Every operation is scoped to the sale's
// tenant.
type SaleRepository interface {
	// SaveConfirmed stores a confirmed sale and calls reserve within the
	// same transaction, so the sale and its stock exits are stored together
	// or not at all. If reserve fails, nothing is stored and its error is
	// returned.
	SaveConfirmed(ctx context.Context, s *Sale, reserve func(context.Context) error) error
}

// PolicyRepository reads the tenant's sales rules.
type PolicyRepository interface {
	// Get returns the tenant's policy, or the zero SalesPolicy if it has not
	// set one.
	Get(ctx context.Context, tenantID uuid.UUID) (SalesPolicy, error)
}
