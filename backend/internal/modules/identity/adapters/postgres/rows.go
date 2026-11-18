package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// readOne reads one row of tenantID with query and converts it to the
// domain; no row is notFound. what names the read in errors.
func readOne[R, E any](ctx context.Context, d *db.DB, tenantID uuid.UUID, what string, notFound error,
	query func(*sqlcgen.Queries) (R, error), convert func(R) (E, error),
) (E, error) {
	var zero E
	row, err := db.Read(ctx, d, tenantID, func(tx pgx.Tx) (R, error) { return query(sqlcgen.New(tx)) })
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return zero, notFound
	case err != nil:
		return zero, fmt.Errorf("identity: %s: %w", what, err)
	}
	return convert(row)
}

// readAll reads the rows of tenantID with query and converts each one to
// the domain. what names the read in errors.
func readAll[R, E any](ctx context.Context, d *db.DB, tenantID uuid.UUID, what string,
	query func(*sqlcgen.Queries) ([]R, error), convert func(R) (E, error),
) ([]E, error) {
	rows, err := db.Read(ctx, d, tenantID, func(tx pgx.Tx) ([]R, error) { return query(sqlcgen.New(tx)) })
	if err != nil {
		return nil, fmt.Errorf("identity: %s: %w", what, err)
	}
	out := make([]E, 0, len(rows))
	for _, row := range rows {
		e, err := convert(row)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}
