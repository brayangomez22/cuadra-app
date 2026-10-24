// Package postgres implements the sales repositories on PostgreSQL with
// sqlc. Every operation runs in a transaction scoped to the tenant.
package postgres

import (
	"fmt"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
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

// pageBounds checks a page (1 to 100 rows) and converts it to the queries'
// int32 parameters.
func pageBounds(limit, offset int) (int32, int32, error) {
	if limit < 1 || limit > maxPageSize {
		return 0, 0, fmt.Errorf("sales: %w (limit %d, offset %d)", db.ErrPageOutOfRange, limit, offset)
	}
	return db.PageBounds(limit, offset)
}
