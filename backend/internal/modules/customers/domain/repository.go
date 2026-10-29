package domain

import (
	"context"

	"github.com/google/uuid"
)

// CustomerStatus filters a search by whether the customers are active.
type CustomerStatus string

// Statuses of a customer search.
const (
	StatusActive   CustomerStatus = "active"
	StatusInactive CustomerStatus = "inactive"
	StatusAll      CustomerStatus = "all"
)

// CustomerFilter is a page of a customer search of a tenant. Text matches
// the name or the trade name (every word, ignoring accents and case) or the
// document number, exact matches first.
type CustomerFilter struct {
	TenantID  uuid.UUID
	Text      string
	SegmentID *uuid.UUID
	Status    CustomerStatus
	Limit     int
	Offset    int
}

// CustomerRepository stores the customers of a tenant. Create and Update
// fail with a *DocumentTakenError if another customer of the tenant has the
// same document, and with ErrSegmentNotFound if the segment is not the
// tenant's.
type CustomerRepository interface {
	Create(ctx context.Context, c *Customer) error
	// Update fails with ErrCustomerNotFound if the customer is not the
	// tenant's.
	Update(ctx context.Context, c *Customer) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error)
	// Names returns the names of the tenant's customers among ids, active
	// or not.
	Names(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]string, error)
	// Search returns a page of the filter and how many customers match it.
	Search(ctx context.Context, f CustomerFilter) ([]*Customer, int, error)
}

// SegmentRepository stores the customer segments of a tenant. Create and
// Update fail with ErrSegmentNameTaken if another segment of the tenant has
// the same name, ignoring case.
type SegmentRepository interface {
	Create(ctx context.Context, s *Segment) error
	Update(ctx context.Context, s *Segment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Segment, error)
	// List returns the tenant's segments by name.
	List(ctx context.Context, tenantID uuid.UUID) ([]*Segment, error)
	// Delete fails with ErrSegmentNotFound or, if any customer (active or
	// not) is in it, ErrSegmentInUse.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
