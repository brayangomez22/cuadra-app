package app_test

import (
	"cmp"
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// store is the in-memory database of the fakes: it keeps snapshots, so a
// change is only seen once a repository saves it.
type store struct {
	settings    map[uuid.UUID]domain.Settings
	resolutions map[uuid.UUID]domain.ResolutionSnapshot
	documents   map[uuid.UUID]domain.DocumentSnapshot
	// enqueued are the documents whose issue was enqueued, in order.
	enqueued []uuid.UUID
}

func newStore() *store {
	return &store{
		settings:    map[uuid.UUID]domain.Settings{},
		resolutions: map[uuid.UUID]domain.ResolutionSnapshot{},
		documents:   map[uuid.UUID]domain.DocumentSnapshot{},
	}
}

type fakeSettings struct{ *store }

func (f fakeSettings) Get(_ context.Context, tenantID uuid.UUID) (domain.Settings, error) {
	if s, ok := f.settings[tenantID]; ok {
		return s, nil
	}
	return domain.DefaultSettings(), nil
}

func (f fakeSettings) Save(_ context.Context, tenantID uuid.UUID, s domain.Settings) error {
	f.settings[tenantID] = s
	return nil
}

type fakeResolutions struct{ *store }

func (f fakeResolutions) Create(_ context.Context, r *domain.Resolution) error {
	f.resolutions[r.ID()] = r.Snapshot()
	return nil
}

func (f fakeResolutions) List(_ context.Context, tenantID uuid.UUID) ([]*domain.Resolution, error) {
	return f.of(tenantID, func(domain.ResolutionSnapshot) bool { return true }), nil
}

func (f fakeResolutions) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	s, ok := f.resolutions[id]
	switch {
	case !ok || s.TenantID != tenantID:
		return domain.ErrResolutionNotFound
	case domain.RestoreResolution(s).Used():
		return domain.ErrResolutionInUse
	}
	delete(f.resolutions, id)
	return nil
}

func (f fakeResolutions) LockKind(_ context.Context, tenantID uuid.UUID, kind domain.Kind) ([]*domain.Resolution, error) {
	return f.of(tenantID, func(s domain.ResolutionSnapshot) bool { return s.Params.Kind == kind }), nil
}

func (f fakeResolutions) SaveNext(_ context.Context, r *domain.Resolution) error {
	f.resolutions[r.ID()] = r.Snapshot()
	return nil
}

func (f fakeResolutions) of(tenantID uuid.UUID, keep func(domain.ResolutionSnapshot) bool) []*domain.Resolution {
	var out []*domain.Resolution
	for _, s := range f.resolutions {
		if s.TenantID == tenantID && keep(s) {
			out = append(out, domain.RestoreResolution(s))
		}
	}
	slices.SortFunc(out, func(a, b *domain.Resolution) int { return cmp.Compare(b.ID().String(), a.ID().String()) })
	return out
}

type fakeDocuments struct{ *store }

func (f fakeDocuments) Add(_ context.Context, d *domain.Document) error {
	f.documents[d.ID()] = d.Snapshot()
	f.enqueued = append(f.enqueued, d.ID())
	return nil
}

func (f fakeDocuments) Get(_ context.Context, tenantID, id uuid.UUID) (*domain.Document, error) {
	s, ok := f.documents[id]
	if !ok || s.TenantID != tenantID {
		return nil, domain.ErrDocumentNotFound
	}
	return domain.RestoreDocument(s), nil
}

func (f fakeDocuments) Save(_ context.Context, d *domain.Document) error {
	f.documents[d.ID()] = d.Snapshot()
	return nil
}

func (f fakeDocuments) Requeue(ctx context.Context, d *domain.Document) error {
	return f.Add(ctx, d)
}

func (f fakeDocuments) List(_ context.Context, tenantID uuid.UUID, filter domain.DocumentFilter) (domain.DocumentPage, error) {
	docs := f.where(tenantID, func(s domain.DocumentSnapshot) bool {
		return (filter.Status == nil || s.Status == *filter.Status) && (filter.SaleID == nil || s.SaleID == *filter.SaleID)
	})
	slices.Reverse(docs)
	page := domain.DocumentPage{Total: len(docs)}
	end := min(len(docs), filter.Offset+filter.Limit)
	if filter.Offset < end {
		page.Documents = docs[filter.Offset:end]
	}
	return page, nil
}

func (f fakeDocuments) ExistsForSale(_ context.Context, tenantID, saleID uuid.UUID) (bool, error) {
	return len(f.where(tenantID, func(s domain.DocumentSnapshot) bool { return s.SaleID == saleID })) > 0, nil
}

func (f fakeDocuments) SentTenants(_ context.Context, limit int) ([]uuid.UUID, error) {
	var tenants []uuid.UUID
	for _, s := range f.documents {
		if s.Status == domain.StatusSent && !slices.Contains(tenants, s.TenantID) {
			tenants = append(tenants, s.TenantID)
		}
	}
	return tenants[:min(limit, len(tenants))], nil
}

func (f fakeDocuments) Sent(_ context.Context, tenantID uuid.UUID, limit int) ([]*domain.Document, error) {
	docs := f.where(tenantID, func(s domain.DocumentSnapshot) bool { return s.Status == domain.StatusSent })
	return docs[:min(limit, len(docs))], nil
}

// where returns the tenant's documents that keep, the oldest first.
func (f fakeDocuments) where(tenantID uuid.UUID, keep func(domain.DocumentSnapshot) bool) []*domain.Document {
	var out []*domain.Document
	for _, s := range f.documents {
		if s.TenantID == tenantID && keep(s) {
			out = append(out, domain.RestoreDocument(s))
		}
	}
	slices.SortFunc(out, func(a, b *domain.Document) int { return cmp.Compare(a.ID().String(), b.ID().String()) })
	return out
}
