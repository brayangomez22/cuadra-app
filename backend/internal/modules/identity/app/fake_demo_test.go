package app_test

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// fakeDemo finds the demo tenants among fakeTenants and purges one by
// removing it and its users.
type fakeDemo struct {
	mu      sync.Mutex
	tenants *fakeTenants
	users   *fakeUsers
	purged  []uuid.UUID
}

func (f *fakeDemo) FindByTemplate(_ context.Context, template string) ([]uuid.UUID, error) {
	f.tenants.mu.Lock()
	defer f.tenants.mu.Unlock()
	var out []uuid.UUID
	for id, t := range f.tenants.byID {
		if t.IsDemo() && t.BusinessTemplate() == template {
			out = append(out, id)
		}
	}
	return out, nil
}

func (f *fakeDemo) Purge(_ context.Context, id uuid.UUID) error {
	f.tenants.mu.Lock()
	t, ok := f.tenants.byID[id]
	f.tenants.mu.Unlock()
	if !ok {
		return domain.ErrTenantNotFound
	}
	if !t.IsDemo() {
		return domain.ErrNotDemoTenant
	}
	f.tenants.remove(id)
	f.users.mu.Lock()
	for uid, u := range f.users.byID {
		if u.TenantID() == id {
			delete(f.users.byID, uid)
		}
	}
	f.users.mu.Unlock()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.purged = append(f.purged, id)
	return nil
}
