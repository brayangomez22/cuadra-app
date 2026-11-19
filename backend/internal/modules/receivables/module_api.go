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

// AgingReport is the aging by customer (see app.AgingReport).
type AgingReport = app.AgingReport

// Balance is what a customer owes (see app.Balance).
type Balance = app.Balance

// allRows is a page as long as any tenant's customers.
const allRows = 1 << 30

// AgingAll returns every customer's aging as of today, in sort's order
// (see app.Service.Aging). It is for the reports module's export, through
// the port internal/server adapts.
func (m *Module) AgingAll(ctx context.Context, tenantID, userID uuid.UUID, sort string) (AgingReport, error) {
	return m.service.Aging(ctx, app.Actor{TenantID: tenantID, UserID: userID}, app.AgingQuery{Sort: sort, Limit: allRows})
}

// Balances returns what each of the customers owes (see
// app.Service.Balances). It is for the customers module's list, through
// the port internal/server adapts.
func (m *Module) Balances(ctx context.Context, tenantID, userID uuid.UUID, customerIDs []uuid.UUID) (map[uuid.UUID]Balance, error) {
	return m.service.Balances(ctx, app.Actor{TenantID: tenantID, UserID: userID}, customerIDs)
}
