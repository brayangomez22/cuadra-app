package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PaymentMethod is how a customer pays what it owes. Credit is not one: a
// debt is not paid with more debt.
type PaymentMethod string

// Payment methods of a payment.
const (
	PaymentCash      PaymentMethod = "cash"
	PaymentCard      PaymentMethod = "card"
	PaymentNequi     PaymentMethod = "nequi"
	PaymentDaviplata PaymentMethod = "daviplata"
	PaymentTransfer  PaymentMethod = "transfer"
)

// ParsePaymentMethod returns the method with code s. Matching is exact.
func ParsePaymentMethod(s string) (PaymentMethod, error) {
	m := PaymentMethod(s)
	switch m {
	case PaymentCash, PaymentCard, PaymentNequi, PaymentDaviplata, PaymentTransfer:
		return m, nil
	}
	return "", ErrInvalidPaymentMethod
}

func (m PaymentMethod) String() string { return string(m) }

// PaymentInput is a payment UserID receives. CashSessionID is the drawer a
// cash payment went into, if any.
type PaymentInput struct {
	ID            uuid.UUID
	Method        PaymentMethod
	Amount        decimal.Decimal
	UserID        uuid.UUID
	CashSessionID *uuid.UUID
	At            time.Time
}

// Allocation is how much of a payment went to a receivable.
type Allocation struct {
	ReceivableID uuid.UUID
	Amount       decimal.Decimal
}

// Payment is money a customer paid towards what it owes, split among its
// receivables. It is immutable.
type Payment struct {
	id            uuid.UUID
	tenantID      uuid.UUID
	customerID    uuid.UUID
	method        PaymentMethod
	amount        decimal.Decimal
	userID        uuid.UUID
	cashSessionID *uuid.UUID
	createdAt     time.Time
	allocations   []Allocation
}

// Collect receives a payment and applies it to the open receivables (all of
// the account's), the one due first first. It cannot exceed the balance, so
// the balance never goes negative. On error nothing changes.
func (a *Account) Collect(in PaymentInput, open []*Receivable) (Payment, error) {
	if err := in.validate(); err != nil {
		return Payment{}, err
	}
	if in.Amount.GreaterThan(a.balance) {
		return Payment{}, ErrPaymentExceedsBalance
	}
	order, err := a.dueFirst(open)
	if err != nil {
		return Payment{}, err
	}
	var allocations []Allocation
	left := in.Amount
	for _, r := range order {
		if !left.IsPositive() {
			break
		}
		taken := r.apply(left)
		left = left.Sub(taken)
		allocations = append(allocations, Allocation{ReceivableID: r.id, Amount: taken})
	}
	a.balance = a.balance.Sub(in.Amount)
	return Payment{
		id: in.ID, tenantID: a.tenantID, customerID: a.customerID, method: in.Method, amount: in.Amount,
		userID: in.UserID, cashSessionID: copyID(in.CashSessionID), createdAt: in.At.UTC(), allocations: allocations,
	}, nil
}

func (in PaymentInput) validate() error {
	if in.ID == uuid.Nil {
		return ErrInvalidPaymentID
	}
	if _, err := ParsePaymentMethod(string(in.Method)); err != nil {
		return err
	}
	if !in.Amount.IsPositive() || !isMoney(in.Amount) {
		return ErrInvalidPaymentAmount
	}
	if in.UserID == uuid.Nil {
		return ErrInvalidUserID
	}
	return nil
}

// dueFirst returns the account's open receivables, the one due first first
// (then the oldest). Their balances must add up to the account's: it fails
// with ErrBalanceMismatch otherwise, before anything changes.
func (a *Account) dueFirst(open []*Receivable) ([]*Receivable, error) {
	sum := decimal.Zero
	order := make([]*Receivable, 0, len(open))
	for _, r := range open {
		if r.customerID != a.customerID || r.tenantID != a.tenantID || r.status != ReceivableOpen {
			return nil, ErrBalanceMismatch
		}
		sum = sum.Add(r.balance)
		order = append(order, r)
	}
	if !sum.Equal(a.balance) {
		return nil, ErrBalanceMismatch
	}
	slices.SortStableFunc(order, func(x, y *Receivable) int {
		if c := x.dueDate.t.Compare(y.dueDate.t); c != 0 {
			return c
		}
		return x.createdAt.Compare(y.createdAt)
	})
	return order, nil
}

func copyID(id *uuid.UUID) *uuid.UUID {
	if id == nil {
		return nil
	}
	c := *id
	return &c
}

// ID returns the payment's ID (UUID v7).
func (p Payment) ID() uuid.UUID { return p.id }

// TenantID returns the tenant it belongs to.
func (p Payment) TenantID() uuid.UUID { return p.tenantID }

// CustomerID returns who paid.
func (p Payment) CustomerID() uuid.UUID { return p.customerID }

// Method returns how it was paid.
func (p Payment) Method() PaymentMethod { return p.method }

// Amount returns how much was paid.
func (p Payment) Amount() decimal.Decimal { return p.amount }

// UserID returns who received it.
func (p Payment) UserID() uuid.UUID { return p.userID }

// CashSessionID returns the drawer a cash payment went into, or nil.
func (p Payment) CashSessionID() *uuid.UUID { return copyID(p.cashSessionID) }

// CreatedAt returns when it was received.
func (p Payment) CreatedAt() time.Time { return p.createdAt }

// Allocations returns how it was split, in the order it was applied.
func (p Payment) Allocations() []Allocation { return append([]Allocation(nil), p.allocations...) }
