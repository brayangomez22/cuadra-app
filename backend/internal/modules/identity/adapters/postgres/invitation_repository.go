package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// InvitationRepository persists invitations in Postgres, within their
// tenant (RLS).
type InvitationRepository struct{ db *db.DB }

// NewInvitationRepository returns an InvitationRepository over d.
func NewInvitationRepository(d *db.DB) *InvitationRepository { return &InvitationRepository{db: d} }

// Create stores a new invitation.
func (r *InvitationRepository) Create(ctx context.Context, inv *domain.Invitation) error {
	s := inv.Snapshot()
	err := r.db.WithTenantTx(ctx, s.TenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateInvitation(ctx, sqlcgen.CreateInvitationParams{
			ID: s.ID, TenantID: s.TenantID, Email: s.Email.String(), Name: s.Name, Role: s.Role.String(),
			InvitedBy: s.InvitedBy, TokenHash: s.Hash, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt,
			AcceptedAt: s.AcceptedAt, RevokedAt: s.RevokedAt,
		})
	})
	if err != nil {
		return fmt.Errorf("identity: create invitation: %w", err)
	}
	return nil
}

// GetByID returns domain.ErrInvitationNotFound when tenantID has no such
// invitation.
func (r *InvitationRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Invitation, error) {
	inv, err := r.get(ctx, tenantID, func(q *sqlcgen.Queries) (sqlcgen.UserInvitation, error) {
		return q.GetInvitation(ctx, sqlcgen.GetInvitationParams{TenantID: tenantID, ID: id})
	})
	if errors.Is(err, domain.ErrLinkNotFound) {
		return nil, domain.ErrInvitationNotFound
	}
	return inv, err
}

// GetByHash returns domain.ErrLinkNotFound when no invitation of tenantID
// has hash.
func (r *InvitationRepository) GetByHash(ctx context.Context, tenantID uuid.UUID, hash []byte) (*domain.Invitation, error) {
	return r.get(ctx, tenantID, func(q *sqlcgen.Queries) (sqlcgen.UserInvitation, error) {
		return q.GetInvitationByHash(ctx, sqlcgen.GetInvitationByHashParams{TenantID: tenantID, TokenHash: hash})
	})
}

func (r *InvitationRepository) get(ctx context.Context, tenantID uuid.UUID, query func(*sqlcgen.Queries) (sqlcgen.UserInvitation, error)) (*domain.Invitation, error) {
	return readOne(ctx, r.db, tenantID, "get invitation", domain.ErrLinkNotFound, query, invitationFromRow)
}

// ListOpen returns the invitations neither accepted nor revoked, newest
// first.
func (r *InvitationRepository) ListOpen(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invitation, error) {
	return readAll(ctx, r.db, tenantID, "list invitations", func(q *sqlcgen.Queries) ([]sqlcgen.UserInvitation, error) {
		return q.ListOpenInvitations(ctx, tenantID)
	}, invitationFromRow)
}

// Close stores the acceptance or revocation of inv; domain.ErrLinkUsed if
// the stored one was already closed.
func (r *InvitationRepository) Close(ctx context.Context, inv *domain.Invitation) error {
	s := inv.Snapshot()
	affected, err := db.Affected(ctx, r.db, s.TenantID, func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).CloseInvitation(ctx, sqlcgen.CloseInvitationParams{
			TenantID: s.TenantID, ID: s.ID, AcceptedAt: s.AcceptedAt, RevokedAt: s.RevokedAt,
		})
	})
	switch {
	case err != nil:
		return fmt.Errorf("identity: close invitation: %w", err)
	case affected == 0:
		return domain.ErrLinkUsed
	}
	return nil
}

// RevokeOpen revokes the open invitations of email in tenantID.
func (r *InvitationRepository) RevokeOpen(ctx context.Context, tenantID uuid.UUID, email domain.Email, now time.Time) error {
	revokedAt := now.UTC()
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).RevokeOpenInvitations(ctx, sqlcgen.RevokeOpenInvitationsParams{
			TenantID: tenantID, Email: email.String(), RevokedAt: &revokedAt,
		})
	})
	if err != nil {
		return fmt.Errorf("identity: revoke open invitations: %w", err)
	}
	return nil
}

func invitationFromRow(row sqlcgen.UserInvitation) (*domain.Invitation, error) {
	email, err := domain.ParseEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf("identity: invitation %s: stored email: %w", row.ID, err)
	}
	inv, err := domain.RehydrateInvitation(domain.InvitationSnapshot{
		ID: row.ID, TenantID: row.TenantID, Email: email, Name: row.Name, Role: domain.Role(row.Role),
		InvitedBy: row.InvitedBy, Hash: row.TokenHash, ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt,
		AcceptedAt: row.AcceptedAt, RevokedAt: row.RevokedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("identity: invitation %s: %w", row.ID, err)
	}
	return inv, nil
}
