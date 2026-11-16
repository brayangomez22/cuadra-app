package app_test

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
)

// TenantsWithExpired lists the tenants with tokens expired before before,
// like the SECURITY DEFINER function, sorted.
func (f *fakeTokens) TenantsWithExpired(_ context.Context, before time.Time, limit int) ([]uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []uuid.UUID
	for _, t := range f.byID {
		if t.ExpiresAt().Before(before) && !slices.Contains(out, t.TenantID()) {
			out = append(out, t.TenantID())
		}
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	return out[:min(limit, len(out))], nil
}

// DeleteExpired deletes the tenant's tokens expired before before.
func (f *fakeTokens) DeleteExpired(_ context.Context, tenantID uuid.UUID, before time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for id, t := range f.byID {
		if t.TenantID() == tenantID && t.ExpiresAt().Before(before) {
			delete(f.byID, id)
			n++
		}
	}
	return n, nil
}
