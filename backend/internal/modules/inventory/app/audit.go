package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// auditStockAdjusted is the audit action of a stock correction
// (docs/auditoria.md); its source tells a manual adjustment from one of an
// approved stock count. The reason is free text and stays in the kardex.
const (
	auditStockAdjusted    = "stock.adjusted"
	adjustmentSourceUser  = "adjustment"
	adjustmentSourceCount = "stock_count"
)

// AuditEntry is what an audited operation of the module did: an action of
// the catalog, the entity and location it is about and its details
// (quantities as decimal strings).
type AuditEntry struct {
	Action     string
	EntityID   uuid.UUID
	LocationID uuid.UUID
	Details    map[string]string
}

// AuditLog records the module's audited operations in the audit module.
// It is called with the ctx of the operation's transaction: its error must
// undo the operation.
type AuditLog interface {
	Record(ctx context.Context, tenantID, userID uuid.UUID, e AuditEntry) error
}

// Transactor runs fn in one transaction of tenantID; every repository and
// port called with fn's ctx joins it.
type Transactor interface {
	InTx(ctx context.Context, tenantID uuid.UUID, fn func(context.Context) error) error
}

// auditAdjustment records an adjustment of quantity (signed, in the base
// unit) of variantID at locationID.
func (s *Service) auditAdjustment(ctx context.Context, a Actor, variantID, locationID uuid.UUID, quantity decimal.Decimal, source string) error {
	return s.Audit.Record(ctx, a.TenantID, a.UserID, AuditEntry{
		Action: auditStockAdjusted, EntityID: variantID, LocationID: locationID,
		Details: map[string]string{"quantity": quantity.String(), "source": source},
	})
}
