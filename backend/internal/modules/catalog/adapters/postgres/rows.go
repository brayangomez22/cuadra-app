package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// entityRows reads one kind of the tenant's entities (units, categories,
// attributes, axes) from its sqlc rows R and rehydrates each one as E.
// single and plural name the kind in errors.
type entityRows[R, E any] struct {
	single, plural string
	notFound       error
	fromRow        func(R) (E, error)
}

// one reads the row get returns; notFound when there is none.
func (e entityRows[R, E]) one(ctx context.Context, d *db.DB, tenantID uuid.UUID, get func(*sqlcgen.Queries) (R, error)) (E, error) {
	row, err := db.Read(ctx, d, tenantID, func(tx pgx.Tx) (R, error) { return get(sqlcgen.New(tx)) })
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		var zero E
		return zero, e.notFound
	case err != nil:
		var zero E
		return zero, fmt.Errorf("catalog: get %s: %w", e.single, err)
	}
	return e.fromRow(row)
}

// all reads the rows list returns, in its order.
func (e entityRows[R, E]) all(ctx context.Context, d *db.DB, tenantID uuid.UUID, list func(*sqlcgen.Queries) ([]R, error)) ([]E, error) {
	rows, err := db.Read(ctx, d, tenantID, func(tx pgx.Tx) ([]R, error) { return list(sqlcgen.New(tx)) })
	if err != nil {
		return nil, fmt.Errorf("catalog: list %s: %w", e.plural, err)
	}
	out := make([]E, 0, len(rows))
	for _, row := range rows {
		entity, err := e.fromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, entity)
	}
	return out, nil
}

// writeOne runs write, which must affect the row of one entity, and maps
// its errors: violation translates the constraint violations it knows (nil
// for the rest) and a write that affects no row fails with notFound.
func (e entityRows[R, E]) writeOne(
	ctx context.Context, d *db.DB, tenantID uuid.UUID, action string,
	violation func(error) error, write func(*sqlcgen.Queries) (int64, error),
) error {
	affected, err := db.Affected(ctx, d, tenantID, func(tx pgx.Tx) (int64, error) { return write(sqlcgen.New(tx)) })
	if err != nil {
		if mapped := violation(err); mapped != nil {
			return mapped
		}
		return fmt.Errorf("catalog: %s %s: %w", action, e.single, err)
	}
	if affected == 0 {
		return e.notFound
	}
	return nil
}

// writeTx runs write in a transaction of tenantID. notFound and the domain
// errors in known pass as they are, undoing the write; the rest are wrapped.
func (e entityRows[R, E]) writeTx(
	ctx context.Context, d *db.DB, tenantID uuid.UUID, action string,
	write func(*sqlcgen.Queries) error, known ...error,
) error {
	err := d.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error { return write(sqlcgen.New(tx)) })
	if err == nil || errors.Is(err, e.notFound) {
		return err
	}
	for _, k := range known {
		if errors.Is(err, k) {
			return err
		}
	}
	return fmt.Errorf("catalog: %s %s: %w", action, e.single, err)
}

// onViolation translates a violation of constraint with the SQLSTATE code
// to mapped.
func onViolation(code, constraint string, mapped error) func(error) error {
	return func(err error) error {
		if db.Violates(err, code, constraint) {
			return mapped
		}
		return nil
	}
}

// onCode translates any violation with the SQLSTATE code to mapped.
func onCode(code string, mapped error) func(error) error {
	return func(err error) error {
		if db.HasCode(err, code) {
			return mapped
		}
		return nil
	}
}

// failIfUsed is inUse when a check found the entity in use.
func failIfUsed(used bool, err error, inUse error) error {
	if err != nil {
		return err
	}
	if used {
		return inUse
	}
	return nil
}
