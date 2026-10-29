package domain

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

// Domain errors of the receivables module. The HTTP adapter maps them to
// status codes.
var (
	ErrInvalidTenantID       = errors.New("receivables: invalid tenant id")
	ErrInvalidCustomerID     = errors.New("receivables: invalid customer id")
	ErrInvalidReceivableID   = errors.New("receivables: invalid receivable id")
	ErrInvalidSaleID         = errors.New("receivables: invalid sale id")
	ErrInvalidPaymentID      = errors.New("receivables: invalid payment id")
	ErrInvalidUserID         = errors.New("receivables: invalid user id")
	ErrInvalidCreditLimit    = errors.New("receivables: invalid credit limit")
	ErrInvalidCreditTerm     = errors.New("receivables: invalid credit term")
	ErrInvalidChargeAmount   = errors.New("receivables: invalid charge amount")
	ErrInvalidPaymentMethod  = errors.New("receivables: invalid payment method")
	ErrInvalidPaymentAmount  = errors.New("receivables: invalid payment amount")
	ErrCreditNotEnabled      = errors.New("receivables: the customer has no credit")
	ErrCreditLimitExceeded   = errors.New("receivables: the sale exceeds the customer's available credit")
	ErrPaymentExceedsBalance = errors.New("receivables: the payment exceeds what the customer owes")
	ErrReceivableHasPayments = errors.New("receivables: the receivable has payments")
	ErrReceivableNotOpen     = errors.New("receivables: the receivable is not open")
	// ErrBalanceMismatch means the account's balance does not match its open
	// receivables: a broken invariant, never a user's mistake.
	ErrBalanceMismatch = errors.New("receivables: account balance does not match its receivables")

	// Errors reported by the repositories and the use cases.
	ErrCustomerNotFound   = errors.New("receivables: customer not found")
	ErrReceivableNotFound = errors.New("receivables: receivable not found")
	// ErrCashSessionRequired means a cash payment needs the receiver's open
	// cash session and there is none.
	ErrCashSessionRequired = errors.New("receivables: an open cash session is required to receive cash")
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
