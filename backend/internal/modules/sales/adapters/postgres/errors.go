// Package postgres implements the sales repositories on PostgreSQL with
// sqlc. Every operation runs in a transaction scoped to the tenant.
package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATE codes the adapter maps to domain errors.
const (
	foreignKeyViolation = "23503"
	uniqueViolation     = "23505"
)

// Constraints of migrations/00008_sales.sql and 00009_cash_sessions.sql the
// adapter maps to domain errors.
const (
	salesLocationFKey         = "sales_location_fkey"
	saleSequencesLocationFKey = "sale_sequences_location_fkey"
	cashSessionsLocationFKey  = "cash_sessions_location_fkey"
	cashSessionsOneOpenKey    = "cash_sessions_one_open_key"
)

// maxPageSize is the contract's largest page.
const maxPageSize = 100

// violates reports whether err violates the constraint named constraint with
// the SQLSTATE code.
func violates(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && pgErr.ConstraintName == constraint
}

// pageBounds checks a page (1 to 100 rows) and converts it to the queries'
// int32 parameters.
func pageBounds(limit, offset int) (int32, int32, error) {
	if limit < 1 || limit > maxPageSize || offset < 0 || offset > 1<<31-1 {
		return 0, 0, fmt.Errorf("sales: page out of range (limit %d, offset %d)", limit, offset)
	}
	return int32(limit), int32(offset), nil
}
