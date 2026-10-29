package ports

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	identitydomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables"
	receivablesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	salesapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	salesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// SalesCredit satisfies salesapp.CreditLedger with the receivables module,
// translating its errors to the sales module's. The receivables module
// needs the sales module's cash drawer, so Receivables is set once both
// exist.
type SalesCredit struct{ Receivables *receivables.Module }

// Charge opens the receivable of a credit sale.
func (c *SalesCredit) Charge(ctx context.Context, ch salesapp.CreditCharge) error {
	err := c.Receivables.Charge(ctx, ch.TenantID, ch.UserID, receivables.ChargeRequest{
		CustomerID: ch.CustomerID, SaleID: ch.SaleID, Amount: ch.Amount, AuthorizedBy: ch.AuthorizedBy,
	})
	var exceeded *receivablesdomain.CreditLimitExceededError
	switch {
	case errors.As(err, &exceeded):
		return &salesdomain.CreditLimitExceededError{Available: exceeded.Available}
	case errors.Is(err, receivablesdomain.ErrCreditNotEnabled):
		return salesdomain.ErrCreditNotEnabled
	}
	return err
}

// CancelSale cancels the receivable of a voided credit sale.
func (c *SalesCredit) CancelSale(ctx context.Context, tenantID, userID, saleID uuid.UUID) error {
	err := c.Receivables.CancelSale(ctx, tenantID, userID, saleID)
	if errors.Is(err, receivablesdomain.ErrReceivableHasPayments) {
		return salesdomain.ErrCreditSaleHasPayments
	}
	return err
}

// SalesAuthorizer satisfies salesapp.CreditAuthorizer with the identity
// module: whoever authorizes a sale over a customer's credit needs
// receivables:override.
type SalesAuthorizer struct{ Identity *identity.Module }

// AuthorizeCredit returns the user of tenantID with the credentials, if it
// may exceed a customer's credit.
func (a SalesAuthorizer) AuthorizeCredit(ctx context.Context, tenantID uuid.UUID, c salesapp.Credentials) (uuid.UUID, error) {
	id, err := a.Identity.AuthorizeAction(ctx, tenantID, c.Email, c.Password, identitydomain.PermReceivablesOverride)
	switch {
	case errors.Is(err, identitydomain.ErrTooManyAttempts):
		return uuid.Nil, salesdomain.ErrTooManyAuthorizationAttempts
	case errors.Is(err, identitydomain.ErrInvalidCredentials), errors.Is(err, identitydomain.ErrNotAuthorized):
		return uuid.Nil, salesdomain.ErrCreditAuthorizationRejected
	}
	return id, err
}
