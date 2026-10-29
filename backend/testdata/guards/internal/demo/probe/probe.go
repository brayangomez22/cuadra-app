// Package probe imports a module, which the demo rule forbids: the
// generator writes through the API.
package probe

import sales "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"

var _ = sales.Total
