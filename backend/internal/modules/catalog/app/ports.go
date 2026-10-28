package app

import (
	"context"

	"github.com/google/uuid"
)

// StockHistory tells whether variants have stock movements. The inventory
// keeps them; internal/server satisfies this port with it.
type StockHistory interface {
	HasMovements(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (bool, error)
}
