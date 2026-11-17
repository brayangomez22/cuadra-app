package ports

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit"
	catalogapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	identityapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	inventoryapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	purchasesapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	salesapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
)

// Each audited module declares its AuditLog port in its app/ package; these
// adapters satisfy them with an Auditor.

// Auditor appends events to the audit module. The role of who did the
// operation comes from the access token in ctx, like the tenant and the
// user; an operation that runs without one (a background job on behalf of
// a user, like an import) reads the user's current role from the identity
// module. Identity is set once that module exists: it is built after the
// catalog, which already records price changes.
type Auditor struct {
	Audit    *audit.Module
	Identity *identity.Module
}

// record appends e to the audit log of tenantID as done by userID.
func (a *Auditor) record(ctx context.Context, tenantID, userID uuid.UUID, e audit.Entry) error {
	if p, ok := auth.PrincipalFrom(ctx); ok && p.Role != "" && p.UserID == userID {
		return a.Audit.Record(ctx, tenantID, userID, p.Role, e)
	}
	if a.Identity == nil {
		return errors.New("ports: audit without the identity module")
	}
	role, err := a.Identity.UserRole(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	return a.Audit.Record(ctx, tenantID, userID, role, e)
}

// SalesAudit satisfies salesapp.AuditLog with the audit module.
type SalesAudit struct{ *Auditor }

// Record appends the sales module's event.
func (a SalesAudit) Record(ctx context.Context, tenantID, userID uuid.UUID, e salesapp.AuditEntry) error {
	return a.record(ctx, tenantID, userID, audit.Entry{
		Action: audit.Action(e.Action), EntityID: &e.EntityID, LocationID: &e.LocationID, Details: e.Details,
	})
}

// CatalogAudit satisfies catalogapp.AuditLog with the audit module.
type CatalogAudit struct{ *Auditor }

// Record appends the catalog module's event.
func (a CatalogAudit) Record(ctx context.Context, tenantID, userID uuid.UUID, e catalogapp.AuditEntry) error {
	return a.record(ctx, tenantID, userID, audit.Entry{Action: audit.Action(e.Action), EntityID: &e.EntityID, Details: e.Details})
}

// InventoryAudit satisfies inventoryapp.AuditLog with the audit module.
type InventoryAudit struct{ *Auditor }

// Record appends the inventory module's event.
func (a InventoryAudit) Record(ctx context.Context, tenantID, userID uuid.UUID, e inventoryapp.AuditEntry) error {
	return a.record(ctx, tenantID, userID, audit.Entry{
		Action: audit.Action(e.Action), EntityID: &e.EntityID, LocationID: &e.LocationID, Details: e.Details,
	})
}

// PurchasesAudit satisfies purchasesapp.AuditLog with the audit module.
type PurchasesAudit struct{ *Auditor }

// Record appends the purchases module's event.
func (a PurchasesAudit) Record(ctx context.Context, tenantID, userID uuid.UUID, e purchasesapp.AuditEntry) error {
	return a.record(ctx, tenantID, userID, audit.Entry{
		Action: audit.Action(e.Action), EntityID: &e.EntityID, LocationID: &e.LocationID, Details: e.Details,
	})
}

// IdentityAudit satisfies identityapp.AuditLog with the audit module.
type IdentityAudit struct{ *Auditor }

// Record appends the identity module's event.
func (a IdentityAudit) Record(ctx context.Context, tenantID, userID uuid.UUID, e identityapp.AuditEntry) error {
	return a.record(ctx, tenantID, userID, audit.Entry{Action: audit.Action(e.Action), EntityID: &e.EntityID, Details: e.Details})
}

// ModulesAudit satisfies modules.AuditLog with the audit module.
type ModulesAudit struct{ *Auditor }

// RecordModuleChange appends module.enabled or module.disabled.
func (a ModulesAudit) RecordModuleChange(ctx context.Context, tenantID uuid.UUID, c modules.Change) error {
	action := audit.ModuleDisabled
	if c.Active {
		action = audit.ModuleEnabled
	}
	return a.record(ctx, tenantID, c.By, audit.Entry{Action: action, Details: map[string]string{"module": c.Module}})
}
