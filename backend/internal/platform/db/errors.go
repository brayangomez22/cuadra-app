package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATEs of the constraint violations repositories translate to domain
// errors.
const (
	UniqueViolation     = "23505"
	ForeignKeyViolation = "23503"
)

// Violates reports whether err violates the constraint named constraint with
// the SQLSTATE code.
func Violates(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && pgErr.ConstraintName == constraint
}

// HasCode reports whether err is a PostgreSQL error with the SQLSTATE code,
// whatever the constraint.
func HasCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
