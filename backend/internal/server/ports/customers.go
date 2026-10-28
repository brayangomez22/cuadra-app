package ports

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers"
	customersdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	salesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// SalesCustomers satisfies salesapp.CustomerDirectory with the customers
// module, translating its errors to the sales module's.
type SalesCustomers struct{ Customers *customers.Module }

// CheckCustomer tells whether a sale of tenantID can be made to customerID.
func (c SalesCustomers) CheckCustomer(ctx context.Context, tenantID, userID, customerID uuid.UUID) error {
	err := c.Customers.CheckCustomer(ctx, tenantID, userID, customerID)
	switch {
	case errors.Is(err, customersdomain.ErrCustomerNotFound):
		return salesdomain.ErrCustomerNotFound
	case errors.Is(err, customersdomain.ErrCustomerInactive):
		return salesdomain.ErrCustomerInactive
	}
	return err
}
