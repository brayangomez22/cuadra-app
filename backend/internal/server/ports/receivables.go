package ports

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers"
	customersdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	receivablesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales"
	salesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// ReceivablesCustomers satisfies receivablesapp.CustomerDirectory with the
// customers module.
type ReceivablesCustomers struct{ Customers *customers.Module }

// CheckCustomer tells whether customerID is a customer of tenantID, active
// or not.
func (c ReceivablesCustomers) CheckCustomer(ctx context.Context, tenantID, userID, customerID uuid.UUID) error {
	err := c.Customers.CustomerExists(ctx, tenantID, userID, customerID)
	if errors.Is(err, customersdomain.ErrCustomerNotFound) {
		return receivablesdomain.ErrCustomerNotFound
	}
	return err
}

// Names returns the names of the customers of tenantID among ids.
func (c ReceivablesCustomers) Names(ctx context.Context, tenantID, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	return c.Customers.CustomerNames(ctx, tenantID, userID, ids)
}

// ReceivablesCash satisfies receivablesapp.CashDrawer with the sales
// module's cash sessions.
type ReceivablesCash struct{ Sales *sales.Module }

// ReceiveCash puts a cash payment in userID's open cash session.
func (c ReceivablesCash) ReceiveCash(ctx context.Context, tenantID, userID uuid.UUID, amount decimal.Decimal, reason string) (uuid.UUID, error) {
	id, err := c.Sales.ReceiveCash(ctx, tenantID, userID, amount, reason)
	if errors.Is(err, salesdomain.ErrCashSessionRequired) {
		return uuid.Nil, receivablesdomain.ErrCashSessionRequired
	}
	return id, err
}
