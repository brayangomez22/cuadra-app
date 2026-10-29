package domain

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// Errors of a credit sale, reported by the receivables module and the
// authorization through the ports of app.
var (
	ErrCreditNotEnabled    = errors.New("sales: the customer has no credit")
	ErrCreditLimitExceeded = errors.New("sales: the sale exceeds the customer's available credit")
	// ErrCreditSaleHasPayments means a credit sale cannot be voided: the
	// customer already paid part of it.
	ErrCreditSaleHasPayments = errors.New("sales: the credit sale has payments")
	// ErrCreditAuthorizationRejected means the credentials given to exceed
	// the credit are wrong or their user may not authorize it.
	ErrCreditAuthorizationRejected  = errors.New("sales: credit authorization rejected")
	ErrTooManyAuthorizationAttempts = errors.New("sales: too many credit authorization attempts")
)

// CreditLimitExceededError is an ErrCreditLimitExceeded that says how much
// credit the customer has left.
type CreditLimitExceededError struct {
	Available decimal.Decimal
}

func (e *CreditLimitExceededError) Error() string {
	return fmt.Sprintf("%v (available %s)", ErrCreditLimitExceeded, e.Available)
}

func (e *CreditLimitExceededError) Unwrap() error { return ErrCreditLimitExceeded }
