package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Read runs read in a transaction of tenantID (WithTenantTx) and returns
// what it read. On error it returns the zero value.
func Read[T any](ctx context.Context, d *DB, tenantID uuid.UUID, read func(pgx.Tx) (T, error)) (T, error) {
	var value T
	err := d.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		value, err = read(tx)
		return err
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return value, nil
}

// Affected runs write in a transaction of tenantID and returns the rows it
// affected, so a repository can tell an update of a missing row (0).
func Affected(ctx context.Context, d *DB, tenantID uuid.UUID, write func(pgx.Tx) (int64, error)) (int64, error) {
	return Read(ctx, d, tenantID, write)
}
