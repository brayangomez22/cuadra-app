package app_test

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// PurgeableFiles filters like the SQL function: a finished status, older
// than before and with its file; it does not apply the domain's rule.
func (f *fakeImports) PurgeableFiles(_ context.Context, before time.Time, limit int) ([]domain.FileRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var refs []domain.FileRef
	for id, s := range f.state.imports {
		at := s.CreatedAt
		if s.FinishedAt != nil {
			at = *s.FinishedAt
		}
		if _, has := f.state.files[id]; has && slices.Contains(domain.FinishedStatuses(), s.Status) && at.Before(before) {
			refs = append(refs, domain.FileRef{TenantID: s.TenantID, ImportID: id})
		}
	}
	return refs[:min(len(refs), limit)], nil
}

func (f *fakeImports) DropFile(_ context.Context, tenantID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.state.imports[id]; !ok || s.TenantID != tenantID {
		return domain.ErrImportNotFound
	}
	delete(f.state.files, id)
	return nil
}

// purged reports whether the import's file was dropped.
func (f *fakeImports) purged(id uuid.UUID) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, has := f.state.files[id]
	return !has
}
