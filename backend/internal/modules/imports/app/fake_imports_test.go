package app_test

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// fakeImports keeps the imports, their files and errors, and the jobs
// enqueued when an import is queued.
type fakeImports struct {
	mu      sync.Mutex
	state   importsState
	enqueue []uuid.UUID
}

type importsState struct {
	imports map[uuid.UUID]domain.ImportSnapshot
	files   map[uuid.UUID][]byte
	errors  map[uuid.UUID][]domain.RowError
}

func newFakeImports() *fakeImports {
	return &fakeImports{state: importsState{
		imports: map[uuid.UUID]domain.ImportSnapshot{},
		files:   map[uuid.UUID][]byte{},
		errors:  map[uuid.UUID][]domain.RowError{},
	}}
}

func (f *fakeImports) Create(_ context.Context, imp *domain.Import, data []byte, errs []domain.RowError) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.imports[imp.ID()] = imp.Snapshot()
	f.state.files[imp.ID()] = slices.Clone(data)
	f.state.errors[imp.ID()] = slices.Clone(errs)
	return nil
}

func (f *fakeImports) Get(_ context.Context, tenantID, id uuid.UUID) (*domain.Import, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.state.imports[id]
	if !ok || s.TenantID != tenantID {
		return nil, domain.ErrImportNotFound
	}
	return domain.RehydrateImport(s)
}

func (f *fakeImports) File(_ context.Context, tenantID, id uuid.UUID) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.state.imports[id]; !ok || s.TenantID != tenantID {
		return nil, domain.ErrImportNotFound
	}
	return slices.Clone(f.state.files[id]), nil
}

func (f *fakeImports) Errors(_ context.Context, tenantID, id uuid.UUID, limit, offset int) ([]domain.RowError, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.state.imports[id]; !ok || s.TenantID != tenantID {
		return nil, 0, domain.ErrImportNotFound
	}
	all := f.state.errors[id]
	end := min(offset+limit, len(all))
	if offset >= len(all) {
		return nil, len(all), nil
	}
	return slices.Clone(all[offset:end]), len(all), nil
}

func (f *fakeImports) Queue(_ context.Context, imp *domain.Import) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.imports[imp.ID()] = imp.Snapshot()
	f.enqueue = append(f.enqueue, imp.ID())
	return nil
}

func (f *fakeImports) Save(_ context.Context, imp *domain.Import, errs []domain.RowError) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state.imports[imp.ID()] = imp.Snapshot()
	f.state.errors[imp.ID()] = append(f.state.errors[imp.ID()], errs...)
	return nil
}

func (f *fakeImports) snapshot() importsState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return importsState{
		imports: maps.Clone(f.state.imports),
		files:   maps.Clone(f.state.files),
		errors:  maps.Clone(f.state.errors),
	}
}

func (f *fakeImports) restore(s importsState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = s
}

// stored returns the import as the repository has it.
func (f *fakeImports) stored(id uuid.UUID) domain.ImportSnapshot {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state.imports[id]
}
