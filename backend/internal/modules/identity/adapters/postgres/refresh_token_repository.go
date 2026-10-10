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

// RefreshTokenRepository implements domain.RefreshTokenRepository.
type RefreshTokenRepository struct{ db *db.DB }

// NewRefreshTokenRepository returns a RefreshTokenRepository on d.
func NewRefreshTokenRepository(d *db.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: d}
}

// Create stores a new token.
func (r *RefreshTokenRepository) Create(ctx context.Context, t *domain.RefreshToken) error {
	err := r.db.WithTenantTx(ctx, t.TenantID(), func(tx pgx.Tx) error {
		return createToken(ctx, sqlcgen.New(tx), t)
	})
	if err != nil {
		return fmt.Errorf("identity: create refresh token: %w", err)
	}
	return nil
}

func createToken(ctx context.Context, q *sqlcgen.Queries, t *domain.RefreshToken) error {
	return q.CreateRefreshToken(ctx, sqlcgen.CreateRefreshTokenParams{
		ID:        t.ID(),
		TenantID:  t.TenantID(),
		UserID:    t.UserID(),
		FamilyID:  t.FamilyID(),
		TokenHash: t.Hash(),
		ExpiresAt: t.ExpiresAt(),
		CreatedAt: t.CreatedAt(),
		RevokedAt: t.RevokedAt(),
	})
}

// GetByHash returns domain.ErrRefreshTokenNotFound when no token of tenantID
// has hash.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tenantID uuid.UUID, hash []byte) (*domain.RefreshToken, error) {
	var row sqlcgen.RefreshToken
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		row, err = sqlcgen.New(tx).GetRefreshTokenByHash(ctx, sqlcgen.GetRefreshTokenByHashParams{TenantID: tenantID, TokenHash: hash})
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrRefreshTokenNotFound
	case err != nil:
		return nil, fmt.Errorf("identity: get refresh token: %w", err)
	}
	t, err := domain.RehydrateRefreshToken(domain.RefreshTokenSnapshot{
		ID:        row.ID,
		TenantID:  row.TenantID,
		UserID:    row.UserID,
		FamilyID:  row.FamilyID,
		Hash:      row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
		RevokedAt: row.RevokedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("identity: refresh token %s: %w", row.ID, err)
	}
	return t, nil
}

// Rotate stores the revocation of old and the new token next in one
// transaction. The revocation is conditional (revoked_at IS NULL), so of two
// concurrent rotations of the same token only one succeeds; the other gets
// domain.ErrRefreshTokenReused and stores nothing.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, old, next *domain.RefreshToken) error {
	revokedAt := old.RevokedAt()
	if revokedAt == nil || old.TenantID() != next.TenantID() {
		return errors.New("identity: rotate: old must be revoked and in next's tenant")
	}
	err := r.db.WithTenantTx(ctx, old.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		affected, err := q.RevokeRefreshToken(ctx, sqlcgen.RevokeRefreshTokenParams{
			TenantID:  old.TenantID(),
			ID:        old.ID(),
			RevokedAt: revokedAt,
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrRefreshTokenReused
		}
		return createToken(ctx, q, next)
	})
	switch {
	case errors.Is(err, domain.ErrRefreshTokenReused):
		return err
	case err != nil:
		return fmt.Errorf("identity: rotate refresh token: %w", err)
	}
	return nil
}

// RevokeFamily revokes every unrevoked token of the family and returns how
// many it revoked.
func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, tenantID, familyID uuid.UUID, now time.Time) (int64, error) {
	var revoked int64
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		revokedAt := now.UTC()
		var err error
		revoked, err = sqlcgen.New(tx).RevokeRefreshTokenFamily(ctx, sqlcgen.RevokeRefreshTokenFamilyParams{
			TenantID:  tenantID,
			FamilyID:  familyID,
			RevokedAt: &revokedAt,
		})
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("identity: revoke refresh token family: %w", err)
	}
	return revoked, nil
}
