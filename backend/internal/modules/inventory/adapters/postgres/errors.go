package postgres

import (
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATEs of the constraint violations the repositories translate.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

// Constraints of migrations/00005_inventory.sql the adapter maps to domain errors.
const (
	locationsNameKey        = "locations_tenant_id_name_key"
	stockLevelsProductFKey  = "stock_levels_product_fkey"
	stockLevelsLocationFKey = "stock_levels_location_fkey"
)

// violates reports whether err violates the constraint named constraint with
// the SQLSTATE code.
func violates(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && pgErr.ConstraintName == constraint
}

// pageBounds converts a page to the queries' int32 parameters.
func pageBounds(limit, offset int) (int32, int32, error) {
	if limit < 0 || limit > math.MaxInt32 || offset < 0 || offset > math.MaxInt32 {
		return 0, 0, fmt.Errorf("inventory: page out of range (limit %d, offset %d)", limit, offset)
	}
	return int32(limit), int32(offset), nil
}
