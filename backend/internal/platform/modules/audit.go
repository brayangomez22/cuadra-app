package modules

import (
	"context"

	"github.com/google/uuid"
)

// AuditLog records a tenant's module changes in its audit log;
// internal/server adapts the audit module. It is called with the ctx of the
// change's transaction: its error must undo the change.
type AuditLog interface {
	RecordModuleChange(ctx context.Context, tenantID uuid.UUID, c Change) error
}

// Transactor runs fn in one transaction of tenantID; the store and the
// audit log called with fn's ctx join it.
type Transactor interface {
	InTx(ctx context.Context, tenantID uuid.UUID, fn func(context.Context) error) error
}
