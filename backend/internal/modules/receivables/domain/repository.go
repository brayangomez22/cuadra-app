package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Change is what a credit operation changed in an account, for the
// repository to store with its new balance: a receivable opened, a payment
// received or the receivables whose balance or status changed.
type Change struct {
	Opened  *Receivable
	Payment *Payment
	Changed []*Receivable
}

// SaleReceivable is the receivable a sale opened: whose it is and its state.
type SaleReceivable struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Status     ReceivableStatus
}

// MovementKind is what moved a customer's balance.
type MovementKind string

// Movement kinds of a statement.
const (
	MovementCharge       MovementKind = "charge"
	MovementPayment      MovementKind = "payment"
	MovementCancellation MovementKind = "cancellation"
)

// Movement is a line of a customer's statement: a credit sale (with its due
// date), a payment (with its method) or a canceled sale, and the balance
// after it.
type Movement struct {
	Kind    MovementKind
	At      time.Time
	Amount  decimal.Decimal
	Balance decimal.Decimal
	// SaleID is set for a charge or a cancellation, PaymentID and Method for
	// a payment, DueDate for a charge.
	SaleID    *uuid.UUID
	PaymentID *uuid.UUID
	Method    PaymentMethod
	DueDate   *Date
}

// OpenBalance is the balance of an open receivable and its due date, for
// the aging.
type OpenBalance struct {
	CustomerID uuid.UUID
	DueDate    Date
	Balance    decimal.Decimal
}

// CreditRepository stores the credit accounts of a tenant, their
// receivables and their payments.
type CreditRepository interface {
	// Account returns the customer's account; a customer without one has no
	// credit and gets a new account with zero terms.
	Account(ctx context.Context, tenantID, customerID uuid.UUID) (*Account, error)
	// SaveTerms stores the account's terms, creating it if needed; its
	// balance is the repository's to keep.
	SaveTerms(ctx context.Context, a *Account, at time.Time) error
	// Update locks the customer's account (in ctx's transaction, if any),
	// calls fn with it and its open receivables, and stores the account's
	// balance and the Change fn returns. If fn fails, nothing is stored.
	Update(ctx context.Context, tenantID, customerID uuid.UUID,
		fn func(ctx context.Context, a *Account, open []*Receivable) (Change, error)) error
	// OfSale returns the receivable saleID opened, or ErrReceivableNotFound.
	OfSale(ctx context.Context, tenantID, saleID uuid.UUID) (SaleReceivable, error)
	// Open returns the customer's open receivables, due first first.
	Open(ctx context.Context, tenantID, customerID uuid.UUID) ([]*Receivable, error)
	// Movements returns every movement of the customer's statement, in any
	// order and without balances (see Statement).
	Movements(ctx context.Context, tenantID, customerID uuid.UUID) ([]Movement, error)
	// OpenBalances returns the balances of every open receivable of the
	// tenant.
	OpenBalances(ctx context.Context, tenantID uuid.UUID) ([]OpenBalance, error)
}
