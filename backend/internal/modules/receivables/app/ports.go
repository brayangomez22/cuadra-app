package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerDirectory reads the customers of the customers module.
type CustomerDirectory interface {
	// CheckCustomer fails with domain.ErrCustomerNotFound if customerID is
	// not a customer of tenantID. An inactive customer still has its credit:
	// it can pay what it owes.
	CheckCustomer(ctx context.Context, tenantID, userID, customerID uuid.UUID) error
	// Names returns the names of the customers of tenantID among ids.
	Names(ctx context.Context, tenantID, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// CashDrawer puts the cash of a payment in the receiver's open cash session,
// in the sales module.
type CashDrawer interface {
	// ReceiveCash records amount as a cash in of userID's open session, for
	// reason, joining ctx's transaction. It returns the session's id, or
	// uuid.Nil if the user has none and the tenant lets users work without
	// one; it fails with domain.ErrCashSessionRequired otherwise.
	ReceiveCash(ctx context.Context, tenantID, userID uuid.UUID, amount decimal.Decimal, reason string) (uuid.UUID, error)
}
