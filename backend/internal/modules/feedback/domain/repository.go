package domain

import (
	"context"

	"github.com/google/uuid"
)

// Repository keeps the comments of each tenant.
type Repository interface {
	// Create stores a comment and queues its notification to the team, in
	// one transaction (outbox): neither exists without the other.
	Create(ctx context.Context, f *Feedback) error
	// Get returns a comment of the tenant; ErrFeedbackMissing if it has none
	// with that ID.
	Get(ctx context.Context, tenantID, id uuid.UUID) (*Feedback, error)
	// MarkNotified stores that the team was told about f.
	MarkNotified(ctx context.Context, f *Feedback) error
}
