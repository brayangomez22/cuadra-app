package app_test

import (
	"bytes"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

type fakeTenants struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*domain.Tenant
	// users receives the owner on CreateWithOwner.
	users *fakeUsers
}

func cloneTenant(t *domain.Tenant) *domain.Tenant {
	c, err := domain.RehydrateTenant(domain.TenantSnapshot{
		ID: t.ID(), Name: t.Name(), NIT: t.NIT(), Status: t.Status(), BusinessTemplate: t.BusinessTemplate(),
		DefaultTaxRate: t.DefaultTaxRate(), Demo: t.IsDemo(), CreatedAt: t.CreatedAt(),
	})
	if err != nil {
		panic(err)
	}
	return c
}

func (f *fakeTenants) Create(_ context.Context, t *domain.Tenant) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[t.ID()] = cloneTenant(t)
	return nil
}

func (f *fakeTenants) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrTenantNotFound
	}
	return cloneTenant(t), nil
}

func (f *fakeTenants) Update(_ context.Context, t *domain.Tenant) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byID[t.ID()]; !ok {
		return domain.ErrTenantNotFound
	}
	f.byID[t.ID()] = cloneTenant(t)
	return nil
}

// inTx marks the context CreateWithOwner passes to setUp, as the real
// repository passes one that carries its transaction.
type inTx struct{}

// CreateWithOwner stores the tenant and the owner, calls setUp and undoes
// both if it fails, as a rollback would.
func (f *fakeTenants) CreateWithOwner(ctx context.Context, t *domain.Tenant, owner *domain.User, setUp func(context.Context) error) error {
	if err := f.Create(ctx, t); err != nil {
		return err
	}
	if err := f.users.Create(ctx, owner); err != nil {
		f.remove(t.ID())
		return err
	}
	if err := setUp(context.WithValue(ctx, inTx{}, t.ID())); err != nil {
		f.remove(t.ID())
		f.users.remove(owner.ID())
		return err
	}
	return nil
}

func (f *fakeTenants) remove(id uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byID, id)
}

func (f *fakeUsers) remove(id uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byID, id)
}

type fakeUsers struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*domain.User
}

func cloneUser(u *domain.User) *domain.User {
	c, err := domain.RehydrateUser(domain.UserSnapshot{
		ID: u.ID(), TenantID: u.TenantID(), Email: u.Email(), Name: u.Name(),
		PasswordHash: u.PasswordHash(), Role: u.Role(), Active: u.IsActive(), CreatedAt: u.CreatedAt(),
	})
	if err != nil {
		panic(err)
	}
	return c
}

func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, other := range f.byID {
		if other.TenantID() == u.TenantID() && other.Email() == u.Email() {
			return domain.ErrEmailTaken
		}
	}
	f.byID[u.ID()] = cloneUser(u)
	return nil
}

func (f *fakeUsers) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok || u.TenantID() != tenantID {
		return nil, domain.ErrUserNotFound
	}
	return cloneUser(u), nil
}

func (f *fakeUsers) GetByEmail(_ context.Context, tenantID uuid.UUID, email domain.Email) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if u.TenantID() == tenantID && u.Email() == email {
			return cloneUser(u), nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeUsers) Update(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.byID[u.ID()]; !ok || old.TenantID() != u.TenantID() {
		return domain.ErrUserNotFound
	}
	f.byID[u.ID()] = cloneUser(u)
	return nil
}

// FindByEmail implements domain.LoginDirectory over the same users.
func (f *fakeUsers) FindByEmail(_ context.Context, email domain.Email) ([]domain.LoginCandidate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.LoginCandidate
	for _, u := range f.byID {
		if u.Email() == email {
			out = append(out, domain.LoginCandidate{TenantID: u.TenantID(), UserID: u.ID()})
		}
	}
	slices.SortFunc(out, func(a, b domain.LoginCandidate) int { return bytes.Compare(a.TenantID[:], b.TenantID[:]) })
	if len(out) > domain.MaxLoginCandidates {
		out = out[:domain.MaxLoginCandidates]
	}
	return out, nil
}

type fakeTokens struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*domain.RefreshToken
	// beforeRotate, when set, runs inside Rotate before the check: it
	// simulates a concurrent rotation of the same token.
	beforeRotate func()
}

func cloneToken(t *domain.RefreshToken) *domain.RefreshToken {
	c, err := domain.RehydrateRefreshToken(domain.RefreshTokenSnapshot{
		ID: t.ID(), TenantID: t.TenantID(), UserID: t.UserID(), FamilyID: t.FamilyID(),
		Hash: t.Hash(), ExpiresAt: t.ExpiresAt(), CreatedAt: t.CreatedAt(), RevokedAt: t.RevokedAt(),
	})
	if err != nil {
		panic(err)
	}
	return c
}

func (f *fakeTokens) Create(_ context.Context, t *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[t.ID()] = cloneToken(t)
	return nil
}

func (f *fakeTokens) GetByHash(_ context.Context, tenantID uuid.UUID, hash []byte) (*domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.byID {
		if t.TenantID() == tenantID && bytes.Equal(t.Hash(), hash) {
			return cloneToken(t), nil
		}
	}
	return nil, domain.ErrRefreshTokenNotFound
}

func (f *fakeTokens) Rotate(_ context.Context, old, next *domain.RefreshToken) error {
	if f.beforeRotate != nil {
		f.beforeRotate()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.byID[old.ID()]
	if !ok || stored.IsRevoked() {
		return domain.ErrRefreshTokenReused
	}
	f.byID[old.ID()] = cloneToken(old)
	f.byID[next.ID()] = cloneToken(next)
	return nil
}

func (f *fakeTokens) RevokeFamily(_ context.Context, tenantID, familyID uuid.UUID, now time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var revoked int64
	for id, t := range f.byID {
		if t.TenantID() != tenantID || t.FamilyID() != familyID || t.IsRevoked() {
			continue
		}
		revokedAt := now
		f.byID[id] = mustRehydrateToken(t, &revokedAt)
		revoked++
	}
	return revoked, nil
}

func mustRehydrateToken(t *domain.RefreshToken, revokedAt *time.Time) *domain.RefreshToken {
	c, err := domain.RehydrateRefreshToken(domain.RefreshTokenSnapshot{
		ID: t.ID(), TenantID: t.TenantID(), UserID: t.UserID(), FamilyID: t.FamilyID(),
		Hash: t.Hash(), ExpiresAt: t.ExpiresAt(), CreatedAt: t.CreatedAt(), RevokedAt: revokedAt,
	})
	if err != nil {
		panic(err)
	}
	return c
}

// family returns the stored tokens of a family.
func (f *fakeTokens) family(familyID uuid.UUID) []*domain.RefreshToken {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.RefreshToken
	for _, t := range f.byID {
		if t.FamilyID() == familyID {
			out = append(out, cloneToken(t))
		}
	}
	return out
}
