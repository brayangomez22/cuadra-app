package receivables

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
)

// ChargeRequest is a credit sale for Charge (see app.ChargeRequest).
type ChargeRequest = app.ChargeRequest

// Charge opens the receivable of a credit sale userID made (see
// app.Service.Charge). It is for the sales module, through the port
// internal/server adapts; with the sale's ctx it joins its transaction.
func (m *Module) Charge(ctx context.Context, tenantID, userID uuid.UUID, in ChargeRequest) error {
	_, err := m.service.Charge(ctx, app.Actor{TenantID: tenantID, UserID: userID}, in)
	return err
}

// CancelSale cancels the receivable of a voided credit sale (see
// app.Service.CancelSale). It is for the sales module; with the void's ctx
// it joins its transaction.
func (m *Module) CancelSale(ctx context.Context, tenantID, userID, saleID uuid.UUID) error {
	return m.service.CancelSale(ctx, app.Actor{TenantID: tenantID, UserID: userID}, saleID)
}
