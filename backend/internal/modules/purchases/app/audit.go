package app

import (
	"context"

	"github.com/google/uuid"
)

// auditPurchaseVoided is the audit action of a void (docs/auditoria.md).
const auditPurchaseVoided = "purchase.voided"

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
