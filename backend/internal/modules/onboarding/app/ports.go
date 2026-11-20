package app

import (
	"context"

	"github.com/google/uuid"
)

// Templates gives the getting-started steps of a business template, in its
// order; ok is false for a template that no longer exists.
type Templates interface {
	Steps(templateID string) (steps []string, ok bool)
}

// Modules gives the tenant's active modules.
type Modules interface {
	Active(ctx context.Context, tenantID uuid.UUID) ([]string, error)
}

// InvoicingReadiness is whether the tenant can issue electronic documents:
// emission turned on and a numbering resolution in force for its default
// kind (the invoicing module's rule).
type InvoicingReadiness struct {
	Enabled bool
	Ready   bool
}

// Invoicing reads the tenant's electronic invoicing readiness.
type Invoicing interface {
	Readiness(ctx context.Context, a Actor) (InvoicingReadiness, error)
}
