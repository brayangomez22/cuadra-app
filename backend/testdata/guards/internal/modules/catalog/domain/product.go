// Package domain imports what the domain rule forbids: an adapter, HTTP and
// another module.
package domain

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres"
	sales "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// Probe uses the imports.
var Probe = []any{http.MethodGet, postgres.Table, sales.Total}
