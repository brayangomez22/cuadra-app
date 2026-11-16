package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// TenantsWithExpired returns, across tenants, up to limit of those with
// refresh tokens expired before before. It reads only their ids, through
// tenants_with_expired_refresh_tokens (00035, SECURITY DEFINER), outside
// any tenant's transaction.
func (r *RefreshTokenRepository) TenantsWithExpired(ctx context.Context, before time.Time, limit int) ([]uuid.UUID, error) {
	maxTenants, _, err := db.PageBounds(limit, 0)
	if err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	err = r.db.WithTx(ctx, func(tx pgx.Tx) error {
		var err error
		ids, err = sqlcgen.New(tx).TenantsWithExpiredRefreshTokens(ctx, sqlcgen.TenantsWithExpiredRefreshTokensParams{
			Before: before, MaxTenants: maxTenants,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("identity: tenants with expired refresh tokens: %w", err)
	}
	return ids, nil
}

// DeleteExpired deletes the tenant's refresh tokens expired before before
// and returns how many.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context, tenantID uuid.UUID, before time.Time) (int64, error) {
	var n int64
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		n, err = sqlcgen.New(tx).DeleteExpiredRefreshTokens(ctx, before)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("identity: delete expired refresh tokens: %w", err)
	}
	return n, nil
}
