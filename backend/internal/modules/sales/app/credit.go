package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// Credentials are the email and password of the user who authorizes a sale
// over the customer's available credit. They are checked and dropped: never
// stored nor logged.
type Credentials struct {
	Email    string
	Password string
}

// CreditCharge is the part of a confirmed sale paid on credit: the seller,
// the customer, the sale and the amount, and who authorized exceeding the
// customer's credit, if anyone did.
type CreditCharge struct {
	TenantID     uuid.UUID
	UserID       uuid.UUID
	CustomerID   uuid.UUID
	SaleID       uuid.UUID
	Amount       decimal.Decimal
	AuthorizedBy *uuid.UUID
}

// CreditLedger opens and cancels the receivables of credit sales in the
// receivables module. Both join ctx's transaction.
type CreditLedger interface {
	// Charge opens the receivable of a credit sale. It fails with
	// domain.ErrCreditNotEnabled if the customer has no credit and with a
	// *domain.CreditLimitExceededError if the amount exceeds its available
	// credit and nobody authorized it.
	Charge(ctx context.Context, c CreditCharge) error
	// CancelSale cancels the receivable of a voided credit sale. It fails
	// with domain.ErrCreditSaleHasPayments if the customer paid part of it.
	CancelSale(ctx context.Context, tenantID, userID, saleID uuid.UUID) error
}

// CreditAuthorizer checks who authorizes a sale over the customer's credit,
// in the identity module.
type CreditAuthorizer interface {
	// AuthorizeCredit returns the active user of tenantID with the
	// credentials, if its role may exceed a customer's credit. It fails with
	// domain.ErrCreditAuthorizationRejected otherwise, and with
	// domain.ErrTooManyAuthorizationAttempts after too many tries.
	AuthorizeCredit(ctx context.Context, tenantID uuid.UUID, c Credentials) (uuid.UUID, error)
}

// authorizeCredit returns who authorizes the sale to exceed the customer's
// credit, when the sale has a credit part and credentials came with it;
// uuid.Nil otherwise. Whether the sale needs the authorization is the receivables
// module's to say: it keeps the authorizer only if it did.
func (s *Service) authorizeCredit(ctx context.Context, a Actor, sale *domain.Sale, c *Credentials) (uuid.UUID, error) {
	if c == nil || sale.CreditAmount().IsZero() {
		return uuid.Nil, nil
	}
	return s.Authorizer.AuthorizeCredit(ctx, a.TenantID, *c)
}

// chargeCredit opens the receivable of the sale's credit part, if any.
func (s *Service) chargeCredit(ctx context.Context, a Actor, sale *domain.Sale, authorizedBy uuid.UUID) error {
	amount := sale.CreditAmount()
	if amount.IsZero() {
		return nil
	}
	charge := CreditCharge{
		TenantID: a.TenantID, UserID: a.UserID, CustomerID: *sale.CustomerID(), SaleID: sale.ID(), Amount: amount,
	}
	if authorizedBy != uuid.Nil {
		charge.AuthorizedBy = &authorizedBy
	}
	return s.Credit.Charge(ctx, charge)
}
