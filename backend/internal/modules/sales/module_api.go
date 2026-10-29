package sales

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
)

// ReceiveCash records cash userID received outside a sale as a cash in of
// their open session (see app.Service.ReceiveCash). It is for the
// receivables module, whose cash payments go into the drawer; with its ctx
// it joins the payment's transaction.
func (m *Module) ReceiveCash(ctx context.Context, tenantID, userID uuid.UUID, amount decimal.Decimal, reason string) (uuid.UUID, error) {
	return m.service.ReceiveCash(ctx, app.Actor{TenantID: tenantID, UserID: userID}, amount, reason)
}
