package db

import (
	"context"
	"strings"

	"github.com/exaring/otelpgx"
)

// sqlcNamePrefix starts the comment sqlc puts on top of each query:
// "-- name: GetTenant :one".
const sqlcNamePrefix = "-- name: "

// querySpanName names a query's span after its sqlc query ("GetTenant :one"),
// so traces tell queries apart instead of showing only "SELECT". Statements
// without that comment keep otelpgx's default, the SQL operation.
func querySpanName(ctx context.Context, stmt string) string {
	for line := range strings.Lines(stmt) {
		name, ok := strings.CutPrefix(line, sqlcNamePrefix)
		if !ok {
			continue
		}
		if name = strings.TrimSpace(name); name != "" {
			return name
		}
		break
	}
	return otelpgx.SQLOperationName(ctx, stmt)
}
