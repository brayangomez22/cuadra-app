package app

import (
	"context"

	"github.com/google/uuid"
)

// The audit actions this module records (docs/auditoria.md).
const (
	auditSaleVoided               = "sale.voided"
	auditSaleDiscountApplied      = "sale.discount_applied"
	auditCashClosedWithDifference = "cash.closed_with_difference"
)

// AuditEntry is what an audited operation of the module did: an action of
// the catalog, the entity and location it is about and its details
// (amounts as decimal strings).
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
