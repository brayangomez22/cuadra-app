package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// PasswordResetRepository persists password resets in Postgres, within
// their tenant (RLS).
type PasswordResetRepository struct{ db *db.DB }

// NewPasswordResetRepository returns a PasswordResetRepository over d.
func NewPasswordResetRepository(d *db.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: d}
}

// Issue spends the user's unused resets and stores rs, in one transaction.
func (r *PasswordResetRepository) Issue(ctx context.Context, rs *domain.PasswordReset) error {
	s := rs.Snapshot()
	err := r.db.WithTenantTx(ctx, s.TenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		usedAt := s.CreatedAt
		err := q.SpendUnusedPasswordResets(ctx, sqlcgen.SpendUnusedPasswordResetsParams{
			TenantID: s.TenantID, UserID: s.UserID, UsedAt: &usedAt,
		})
		if err != nil {
			return err
		}
		return q.CreatePasswordReset(ctx, sqlcgen.CreatePasswordResetParams{
			ID: s.ID, TenantID: s.TenantID, UserID: s.UserID, IssuedBy: s.IssuedBy, TokenHash: s.Hash,
			ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt, UsedAt: s.UsedAt,
		})
	})
	if err != nil {
		return fmt.Errorf("identity: issue password reset: %w", err)
	}
	return nil
}

// GetByHash returns domain.ErrLinkNotFound when no reset of tenantID has
// hash.
func (r *PasswordResetRepository) GetByHash(ctx context.Context, tenantID uuid.UUID, hash []byte) (*domain.PasswordReset, error) {
	return readOne(ctx, r.db, tenantID, "get password reset", domain.ErrLinkNotFound,
		func(q *sqlcgen.Queries) (sqlcgen.PasswordReset, error) {
			return q.GetPasswordResetByHash(ctx, sqlcgen.GetPasswordResetByHashParams{TenantID: tenantID, TokenHash: hash})
		}, passwordResetFromRow)
}

func passwordResetFromRow(row sqlcgen.PasswordReset) (*domain.PasswordReset, error) {
	rs, err := domain.RehydratePasswordReset(domain.PasswordResetSnapshot{
		ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, IssuedBy: row.IssuedBy, Hash: row.TokenHash,
		ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt, UsedAt: row.UsedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("identity: password reset %s: %w", row.ID, err)
	}
	return rs, nil
}

// MarkUsed stores the use of rs; domain.ErrLinkUsed if it was already used.
func (r *PasswordResetRepository) MarkUsed(ctx context.Context, rs *domain.PasswordReset) error {
	s := rs.Snapshot()
	affected, err := db.Affected(ctx, r.db, s.TenantID, func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).MarkPasswordResetUsed(ctx, sqlcgen.MarkPasswordResetUsedParams{
			TenantID: s.TenantID, ID: s.ID, UsedAt: s.UsedAt,
		})
	})
	switch {
	case err != nil:
		return fmt.Errorf("identity: mark password reset used: %w", err)
	case affected == 0:
		return domain.ErrLinkUsed
	}
	return nil
}
