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

// List returns every user of tenantID, by name.
func (r *UserRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.User, error) {
	return readAll(ctx, r.db, tenantID, "list users", func(q *sqlcgen.Queries) ([]sqlcgen.User, error) {
		return q.ListUsers(ctx, tenantID)
	}, userFromRow)
}

// LockActiveOwners counts the active owners and locks them until the
// transaction of ctx ends; outside one, the lock ends with the count.
func (r *UserRepository) LockActiveOwners(ctx context.Context, tenantID uuid.UUID) (int, error) {
	ids, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]uuid.UUID, error) {
		return sqlcgen.New(tx).LockActiveOwners(ctx, tenantID)
	})
	if err != nil {
		return 0, fmt.Errorf("identity: lock active owners: %w", err)
	}
	return len(ids), nil
}
