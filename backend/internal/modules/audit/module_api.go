package audit

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
)

// Entry is what an audited operation did (see app.Entry).
type Entry = app.Entry

// Action is an action of the audit catalog (see domain.Action).
type Action = domain.Action

// The catalog's actions, for the ports of internal/server.
const (
	SaleVoided               = domain.ActionSaleVoided
	SaleDiscountApplied      = domain.ActionSaleDiscountApplied
	VariantPriceChanged      = domain.ActionVariantPriceChanged
	StockAdjusted            = domain.ActionStockAdjusted
	CashClosedWithDifference = domain.ActionCashClosedWithDifference
	PurchaseVoided           = domain.ActionPurchaseVoided
	ModuleEnabled            = domain.ActionModuleEnabled
	ModuleDisabled           = domain.ActionModuleDisabled
	TenantSettingsChanged    = domain.ActionTenantSettingsChanged
)

// Record appends e to the audit log of tenantID as done by userID with
// role. It is for the other modules, through the ports internal/server
// adapts: with a ctx from db.InTenantTx it joins the audited operation's
// transaction, and its error must undo the operation.
func (m *Module) Record(ctx context.Context, tenantID, userID uuid.UUID, role string, e Entry) error {
	return m.service.Record(ctx, app.Actor{TenantID: tenantID, UserID: userID, Role: role}, e)
}
