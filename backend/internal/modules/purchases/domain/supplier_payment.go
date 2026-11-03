package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PaymentMethod is how the tenant paid a supplier. Paying does not touch the
// cash drawer: money taken from it is recorded there as an outflow
// (docs/decisiones.md, 2026-10-30).
type PaymentMethod string

// Payment methods of a payment to a supplier.
const (
	MethodCash      PaymentMethod = "cash"
	MethodCard      PaymentMethod = "card"
	MethodNequi     PaymentMethod = "nequi"
	MethodDaviplata PaymentMethod = "daviplata"
	MethodTransfer  PaymentMethod = "transfer"
)

// ParsePaymentMethod returns the method with code s. Matching is exact.
func ParsePaymentMethod(s string) (PaymentMethod, error) {
	m := PaymentMethod(s)
	if !m.valid() {
		return "", ErrInvalidPaymentMethod
	}
	return m, nil
}

func (m PaymentMethod) valid() bool {
	switch m {
	case MethodCash, MethodCard, MethodNequi, MethodDaviplata, MethodTransfer:
		return true
	}
	return false
}

// PaymentInput is a payment to the supplier of a purchase, made on PaidOn
// and recorded by UserID.
type PaymentInput struct {
	Method PaymentMethod
	Amount decimal.Decimal
	PaidOn Date
	UserID uuid.UUID
}

// SupplierPayment is money the tenant paid towards a purchase on credit. It
// is immutable and keeps what was left to pay after it.
type SupplierPayment struct {
	id, tenantID, purchaseID, supplierID uuid.UUID
	method                               PaymentMethod
	amount, balanceAfter                 decimal.Decimal
	paidOn                               Date
	userID                               uuid.UUID
	createdAt                            time.Time
}

// Pay records a payment to the supplier and lowers what is left to pay. It
// cannot exceed it, so the balance never goes negative (a purchase in cash
// owes nothing), nor be dated after today in America/Bogota or before the
// invoice. A voided purchase is not paid. On error nothing changes.
func (p *Purchase) Pay(in PaymentInput, now time.Time) (*SupplierPayment, error) {
	if p.voided != nil {
		return nil, ErrPurchaseVoided
	}
	if err := in.check(p.issuedOn, now); err != nil {
		return nil, err
	}
	if in.Amount.GreaterThan(p.balanceDue) {
		return nil, ErrPaymentExceedsBalance
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	p.balanceDue = p.balanceDue.Sub(in.Amount)
	return &SupplierPayment{
		id: id, tenantID: p.tenantID, purchaseID: p.id, supplierID: p.supplierID, method: in.Method,
		amount: in.Amount, balanceAfter: p.balanceDue, paidOn: in.PaidOn, userID: in.UserID, createdAt: now.UTC(),
	}, nil
}

func (in PaymentInput) check(issuedOn Date, now time.Time) error {
	switch {
	case !in.Method.valid():
		return ErrInvalidPaymentMethod
	case !in.Amount.IsPositive() || !in.Amount.Equal(roundMoney(in.Amount)) || !fitsStorage(in.Amount):
		return ErrInvalidPaymentAmount
	case in.PaidOn.IsZero() || DateIn(now).Before(in.PaidOn):
		return ErrPaymentDateInFuture
	case in.PaidOn.Before(issuedOn):
		return ErrPaymentBeforeInvoice
	case in.UserID == uuid.Nil:
		return ErrInvalidUserID
	}
	return nil
}

// ID returns the payment ID (UUID v7).
func (s *SupplierPayment) ID() uuid.UUID { return s.id }

// TenantID returns the tenant the payment belongs to.
func (s *SupplierPayment) TenantID() uuid.UUID { return s.tenantID }

// PurchaseID returns the purchase it pays.
func (s *SupplierPayment) PurchaseID() uuid.UUID { return s.purchaseID }

// SupplierID returns the supplier it was paid to.
func (s *SupplierPayment) SupplierID() uuid.UUID { return s.supplierID }

// Method returns how it was paid.
func (s *SupplierPayment) Method() PaymentMethod { return s.method }

// Amount returns how much was paid.
func (s *SupplierPayment) Amount() decimal.Decimal { return s.amount }

// BalanceAfter returns what was left to pay of the purchase after it.
func (s *SupplierPayment) BalanceAfter() decimal.Decimal { return s.balanceAfter }

// PaidOn returns the day it was paid.
func (s *SupplierPayment) PaidOn() Date { return s.paidOn }

// UserID returns who recorded it.
func (s *SupplierPayment) UserID() uuid.UUID { return s.userID }

// CreatedAt returns when it was recorded, in UTC.
func (s *SupplierPayment) CreatedAt() time.Time { return s.createdAt }
