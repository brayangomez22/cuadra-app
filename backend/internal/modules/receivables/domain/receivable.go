package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReceivableStatus is the state of a receivable.
type ReceivableStatus string

// Receivable states: open while something is owed, paid once its balance
// reaches zero, canceled when its sale is voided before any payment.
const (
	ReceivableOpen     ReceivableStatus = "open"
	ReceivablePaid     ReceivableStatus = "paid"
	ReceivableCanceled ReceivableStatus = "canceled"
)

func (s ReceivableStatus) String() string { return string(s) }

// Authorization is who let a sale exceed the customer's credit, and when.
type Authorization struct {
	UserID uuid.UUID
	At     time.Time
}

// Receivable is what a customer owes for a credit sale: the amount sold on
// credit, what is left to pay and the day it is due. Payments lower its
// balance; it is never negative.
type Receivable struct {
	id            uuid.UUID
	tenantID      uuid.UUID
	customerID    uuid.UUID
	saleID        uuid.UUID
	amount        decimal.Decimal
	balance       decimal.Decimal
	dueDate       Date
	status        ReceivableStatus
	authorization *Authorization
	createdAt     time.Time
	canceledAt    *time.Time
}

// ID returns the receivable's ID (UUID v7).
func (r *Receivable) ID() uuid.UUID { return r.id }

// TenantID returns the tenant it belongs to.
func (r *Receivable) TenantID() uuid.UUID { return r.tenantID }

// CustomerID returns who owes it.
func (r *Receivable) CustomerID() uuid.UUID { return r.customerID }

// SaleID returns the credit sale that opened it.
func (r *Receivable) SaleID() uuid.UUID { return r.saleID }

// Amount returns what was sold on credit.
func (r *Receivable) Amount() decimal.Decimal { return r.amount }

// Balance returns what is left to pay.
func (r *Receivable) Balance() decimal.Decimal { return r.balance }

// DueDate returns the day it is due, in America/Bogota.
func (r *Receivable) DueDate() Date { return r.dueDate }

// Status returns its state.
func (r *Receivable) Status() ReceivableStatus { return r.status }

// CreatedAt returns when the sale opened it.
func (r *Receivable) CreatedAt() time.Time { return r.createdAt }

// Authorization returns who let the sale exceed the credit, or nil if it
// was within it.
func (r *Receivable) Authorization() *Authorization {
	if r.authorization == nil {
		return nil
	}
	a := *r.authorization
	return &a
}

// CanceledAt returns when it was canceled, or nil.
func (r *Receivable) CanceledAt() *time.Time {
	if r.canceledAt == nil {
		return nil
	}
	t := *r.canceledAt
	return &t
}

// hasPayments reports whether any payment went to it.
func (r *Receivable) hasPayments() bool { return !r.balance.Equal(r.amount) }

// apply pays up to amount of its balance and returns how much it took.
func (r *Receivable) apply(amount decimal.Decimal) decimal.Decimal {
	taken := decimal.Min(amount, r.balance)
	r.balance = r.balance.Sub(taken)
	if r.balance.IsZero() {
		r.status = ReceivablePaid
	}
	return taken
}
