package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InvitationRepository persists invitations, scoped to their tenant.
type InvitationRepository interface {
	// Create stores a new invitation.
	Create(ctx context.Context, inv *Invitation) error
	// GetByID returns ErrInvitationNotFound when tenantID has no such one.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Invitation, error)
	// GetByHash returns ErrLinkNotFound when no invitation of tenantID has
	// hash.
	GetByHash(ctx context.Context, tenantID uuid.UUID, hash []byte) (*Invitation, error)
	// ListOpen returns the invitations neither accepted nor revoked
	// (pending or expired), newest first.
	ListOpen(ctx context.Context, tenantID uuid.UUID) ([]*Invitation, error)
	// Close stores the acceptance or revocation of inv. It returns
	// ErrLinkUsed, storing nothing, if the stored one was already accepted
	// or revoked (a concurrent use won).
	Close(ctx context.Context, inv *Invitation) error
	// RevokeOpen revokes the open invitations of email in tenantID, when it
	// is invited again.
	RevokeOpen(ctx context.Context, tenantID uuid.UUID, email Email, now time.Time) error
}

// PasswordResetRepository persists password resets, scoped to their tenant.
type PasswordResetRepository interface {
	// Issue stores r and spends the user's earlier unused resets: only the
	// last link works.
	Issue(ctx context.Context, r *PasswordReset) error
	// GetByHash returns ErrLinkNotFound when no reset of tenantID has hash.
	GetByHash(ctx context.Context, tenantID uuid.UUID, hash []byte) (*PasswordReset, error)
	// MarkUsed stores the use of r. It returns ErrLinkUsed, storing
	// nothing, if it was already used (a concurrent use won).
	MarkUsed(ctx context.Context, r *PasswordReset) error
}
