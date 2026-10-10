// Package migrations embeds the goose SQL migrations so cmd/migrate and the
// test helpers apply exactly the same files.
package migrations

import "embed"

// FS holds the migration files (NNNNN_description.sql).
//
//go:embed *.sql
var FS embed.FS
