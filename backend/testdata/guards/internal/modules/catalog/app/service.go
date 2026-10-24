// Package app imports what the app rule forbids: an adapter and HTTP.
package app

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres"
)

// Probe uses the imports.
var Probe = []any{http.MethodGet, postgres.Table}
