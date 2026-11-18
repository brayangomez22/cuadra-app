package app_test

import (
	"bytes"
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func (f *fakeUsers) List(_ context.Context, tenantID uuid.UUID) ([]*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.User
	for _, u := range f.byID {
		if u.TenantID() == tenantID {
			out = append(out, cloneUser(u))
		}
	}
	slices.SortFunc(out, func(a, b *domain.User) int { return cmp.Compare(a.Name(), b.Name()) })
	return out, nil
}

func (f *fakeUsers) LockActiveOwners(_ context.Context, tenantID uuid.UUID) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, u := range f.byID {
		if u.TenantID() == tenantID && u.Role() == domain.RoleOwner && u.IsActive() {
			n++
		}
	}
	return n, nil
}

func (f *fakeTokens) RevokeUser(_ context.Context, tenantID, userID uuid.UUID, now time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var revoked int64
	for id, t := range f.byID {
		if t.TenantID() != tenantID || t.UserID() != userID || t.IsRevoked() {
			continue
		}
		revokedAt := now
		f.byID[id] = mustRehydrateToken(t, &revokedAt)
		revoked++
	}
	return revoked, nil
}

// live returns how many unrevoked tokens the user has.
func (f *fakeTokens) live(userID uuid.UUID) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, t := range f.byID {
		if t.UserID() == userID && !t.IsRevoked() {
			n++
		}
	}
	return n
}

type fakeInvitations struct {
	mu   sync.Mutex
	byID map[uuid.UUID]domain.InvitationSnapshot
}

func rehydrateInvitation(s domain.InvitationSnapshot) *domain.Invitation {
	inv, err := domain.RehydrateInvitation(s)
	if err != nil {
		panic(err)
	}
	return inv
}

func (f *fakeInvitations) Create(_ context.Context, inv *domain.Invitation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[inv.ID()] = inv.Snapshot()
	return nil
}

func (f *fakeInvitations) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Invitation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byID[id]
	if !ok || s.TenantID != tenantID {
		return nil, domain.ErrInvitationNotFound
	}
	return rehydrateInvitation(s), nil
}

func (f *fakeInvitations) GetByHash(_ context.Context, tenantID uuid.UUID, hash []byte) (*domain.Invitation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.byID {
		if s.TenantID == tenantID && bytes.Equal(s.Hash, hash) {
			return rehydrateInvitation(s), nil
		}
	}
	return nil, domain.ErrLinkNotFound
}

func (f *fakeInvitations) ListOpen(_ context.Context, tenantID uuid.UUID) ([]*domain.Invitation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.Invitation
	for _, s := range f.byID {
		if s.TenantID == tenantID && s.AcceptedAt == nil && s.RevokedAt == nil {
			out = append(out, rehydrateInvitation(s))
		}
	}
	slices.SortFunc(out, func(a, b *domain.Invitation) int { return b.CreatedAt().Compare(a.CreatedAt()) })
	return out, nil
}

func (f *fakeInvitations) Close(_ context.Context, inv *domain.Invitation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.byID[inv.ID()]
	if !ok || stored.AcceptedAt != nil || stored.RevokedAt != nil {
		return domain.ErrLinkUsed
	}
	f.byID[inv.ID()] = inv.Snapshot()
	return nil
}

func (f *fakeInvitations) RevokeOpen(_ context.Context, tenantID uuid.UUID, email domain.Email, now time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, s := range f.byID {
		if s.TenantID == tenantID && s.Email == email && s.AcceptedAt == nil && s.RevokedAt == nil {
			at := now
			s.RevokedAt = &at
			f.byID[id] = s
		}
	}
	return nil
}

type fakeResets struct {
	mu   sync.Mutex
	byID map[uuid.UUID]domain.PasswordResetSnapshot
}

func (f *fakeResets) Issue(_ context.Context, r *domain.PasswordReset) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, s := range f.byID {
		if s.TenantID == r.TenantID() && s.UserID == r.UserID() && s.UsedAt == nil {
			at := r.Snapshot().CreatedAt
			s.UsedAt = &at
			f.byID[id] = s
		}
	}
	f.byID[r.ID()] = r.Snapshot()
	return nil
}

func (f *fakeResets) GetByHash(_ context.Context, tenantID uuid.UUID, hash []byte) (*domain.PasswordReset, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.byID {
		if s.TenantID == tenantID && bytes.Equal(s.Hash, hash) {
			r, err := domain.RehydratePasswordReset(s)
			if err != nil {
				panic(err)
			}
			return r, nil
		}
	}
	return nil, domain.ErrLinkNotFound
}

func (f *fakeResets) MarkUsed(_ context.Context, r *domain.PasswordReset) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.byID[r.ID()]
	if !ok || stored.UsedAt != nil {
		return domain.ErrLinkUsed
	}
	f.byID[r.ID()] = r.Snapshot()
	return nil
}
