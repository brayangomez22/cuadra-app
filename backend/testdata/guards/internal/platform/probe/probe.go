// Package probe imports a module, which the platform rule forbids.
package probe

import sales "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"

// Probe uses the import.
var Probe = sales.Total
