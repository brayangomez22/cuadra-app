package domain

import "github.com/shopspring/decimal"

// PaymentMethod is how a customer pays.
type PaymentMethod string

// Payment methods. Only cash gives change. Credit is the part of the sale
// the customer pays later: it opens a receivable in the receivables module.
const (
	PaymentCash      PaymentMethod = "cash"
	PaymentCard      PaymentMethod = "card"
	PaymentNequi     PaymentMethod = "nequi"
	PaymentDaviplata PaymentMethod = "daviplata"
	PaymentTransfer  PaymentMethod = "transfer"
	PaymentCredit    PaymentMethod = "credit"
)

// ParsePaymentMethod returns the method with code s. Matching is exact.
func ParsePaymentMethod(s string) (PaymentMethod, error) {
	m := PaymentMethod(s)
	switch m {
	case PaymentCash, PaymentCard, PaymentNequi, PaymentDaviplata, PaymentTransfer, PaymentCredit:
		return m, nil
	}
	return "", ErrInvalidPaymentMethod
}

func (m PaymentMethod) String() string { return string(m) }

// Payment is an amount paid with one method. A sale can have several.
type Payment struct {
	method PaymentMethod
	amount decimal.Decimal
}

// NewPayment creates a payment of a positive amount of COP (at most two
// decimals).
func NewPayment(method PaymentMethod, amount decimal.Decimal) (Payment, error) {
	if _, err := ParsePaymentMethod(string(method)); err != nil {
		return Payment{}, err
	}
	if !amount.IsPositive() || !isMoney(amount) {
		return Payment{}, ErrInvalidPaymentAmount
	}
	return Payment{method: method, amount: amount}, nil
}

// Method returns how it was paid.
func (p Payment) Method() PaymentMethod { return p.method }

// Amount returns the amount tendered; for cash it includes the change.
func (p Payment) Amount() decimal.Decimal { return p.amount }
