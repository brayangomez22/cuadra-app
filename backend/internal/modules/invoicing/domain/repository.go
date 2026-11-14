package domain

import (
	"context"

	"github.com/google/uuid"
)

// SettingsRepository stores the tenant's settings.
type SettingsRepository interface {
	// Get returns the tenant's settings, or DefaultSettings if it never
	// changed them.
	Get(ctx context.Context, tenantID uuid.UUID) (Settings, error)
	Save(ctx context.Context, tenantID uuid.UUID, s Settings) error
}

// ResolutionRepository stores the numbering resolutions.
type ResolutionRepository interface {
	Create(ctx context.Context, r *Resolution) error
	// List returns the tenant's resolutions, the newest first.
	List(ctx context.Context, tenantID uuid.UUID) ([]*Resolution, error)
	// Delete removes a resolution that numbered nothing:
	// ErrResolutionNotFound or ErrResolutionInUse.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// LockKind returns the tenant's resolutions of kind, locked until the
	// transaction of ctx ends, so two sales never draw the same number.
	LockKind(ctx context.Context, tenantID uuid.UUID, kind Kind) ([]*Resolution, error)
	// SaveNext stores the resolution's next consecutive, in the
	// transaction of ctx.
	SaveNext(ctx context.Context, r *Resolution) error
}

// DocumentFilter selects a page of documents; nil fields do not filter.
type DocumentFilter struct {
	Status        *Status
	SaleID        *uuid.UUID
	Limit, Offset int
}

// DocumentPage is a page of documents, the newest first, and how many match.
type DocumentPage struct {
	Documents []*Document
	Total     int
}

// DocumentRepository stores the electronic documents.
type DocumentRepository interface {
	// Add stores a new document and enqueues its issue in the same
	// transaction (outbox): with the sale's ctx, in the sale's.
	Add(ctx context.Context, d *Document) error
	// Get returns a document of the tenant, or ErrDocumentNotFound.
	Get(ctx context.Context, tenantID, id uuid.UUID) (*Document, error)
	// Save stores the document's state.
	Save(ctx context.Context, d *Document) error
	// Requeue stores the document's state and enqueues its issue again, in
	// one transaction.
	Requeue(ctx context.Context, d *Document) error
	List(ctx context.Context, tenantID uuid.UUID, f DocumentFilter) (DocumentPage, error)
	// ExistsForSale is whether the sale has a document.
	ExistsForSale(ctx context.Context, tenantID, saleID uuid.UUID) (bool, error)
	// SentTenants returns, across tenants, those with documents sent and
	// waiting for the DIAN.
	SentTenants(ctx context.Context, limit int) ([]uuid.UUID, error)
	// Sent returns the tenant's sent documents, the oldest first.
	Sent(ctx context.Context, tenantID uuid.UUID, limit int) ([]*Document, error)
}
