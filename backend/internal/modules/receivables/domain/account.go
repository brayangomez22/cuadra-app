package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Account is a customer's credit: its terms and what it owes, the sum of
// the balances of its open receivables. Every credit sale, payment and
// cancellation goes through it, so the balance and the receivables change
// together.
type Account struct {
	tenantID   uuid.UUID
	customerID uuid.UUID
	terms      Terms
	balance    decimal.Decimal
}

// NewAccount creates the credit account of a customer, owing nothing.
func NewAccount(tenantID, customerID uuid.UUID, terms Terms) (*Account, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if customerID == uuid.Nil {
		return nil, ErrInvalidCustomerID
	}
	return &Account{tenantID: tenantID, customerID: customerID, terms: terms, balance: decimal.Zero}, nil
}

// TenantID returns the tenant it belongs to.
func (a *Account) TenantID() uuid.UUID { return a.tenantID }

// CustomerID returns whose account it is.
func (a *Account) CustomerID() uuid.UUID { return a.customerID }

// Terms returns its limit and term.
func (a *Account) Terms() Terms { return a.terms }

// Balance returns what the customer owes.
func (a *Account) Balance() decimal.Decimal { return a.balance }

// Available returns how much more the customer may buy on credit: the limit
// minus the balance, never less than zero.
func (a *Account) Available() decimal.Decimal {
	return decimal.Max(a.terms.limit.Sub(a.balance), decimal.Zero)
}

// SetTerms changes the limit and the term. A limit below the balance is
// allowed: the customer just cannot buy more on credit. The term applies to
// the next sales; the receivables keep their due dates.
func (a *Account) SetTerms(t Terms) { a.terms = t }

// ChargeInput is a credit sale: its receivable's ID, the sale, the amount
// sold on credit, when it was sold and, if it exceeds the available credit,
// who authorized it.
type ChargeInput struct {
	ID            uuid.UUID
	SaleID        uuid.UUID
	Amount        decimal.Decimal
	At            time.Time
	Authorization *Authorization
}

// Charge opens the receivable of a credit sale, due the term's days after
// the day of the sale in America/Bogota, and adds it to the balance.
func (a *Account) Charge(in ChargeInput) (*Receivable, error) {
	if in.ID == uuid.Nil {
		return nil, ErrInvalidReceivableID
	}
	if in.SaleID == uuid.Nil {
		return nil, ErrInvalidSaleID
	}
	if !in.Amount.IsPositive() || !isMoney(in.Amount) {
		return nil, ErrInvalidChargeAmount
	}
	if !a.terms.Enabled() {
		return nil, ErrCreditNotEnabled
	}
	var auth *Authorization
	if available := a.Available(); in.Amount.GreaterThan(available) {
		if in.Authorization == nil || in.Authorization.UserID == uuid.Nil {
			return nil, &CreditLimitExceededError{Available: available}
		}
		auth = &Authorization{UserID: in.Authorization.UserID, At: in.Authorization.At.UTC()}
	}
	a.balance = a.balance.Add(in.Amount)
	return &Receivable{
		id: in.ID, tenantID: a.tenantID, customerID: a.customerID, saleID: in.SaleID,
		amount: in.Amount, balance: in.Amount, dueDate: DateIn(in.At).AddDays(a.terms.days),
		status: ReceivableOpen, authorization: auth, createdAt: in.At.UTC(),
	}, nil
}

// Cancel cancels the receivable of a voided sale and takes its balance off
// the account. A receivable with payments cannot be canceled: they would
// have to be given back (credit notes, T37).
func (a *Account) Cancel(r *Receivable, at time.Time) error {
	if r.customerID != a.customerID || r.tenantID != a.tenantID {
		return ErrReceivableNotFound
	}
	if r.status != ReceivableOpen {
		return ErrReceivableNotOpen
	}
	if r.hasPayments() {
		return ErrReceivableHasPayments
	}
	canceledAt := at.UTC()
	a.balance = a.balance.Sub(r.balance)
	r.balance = decimal.Zero
	r.status = ReceivableCanceled
	r.canceledAt = &canceledAt
	return nil
}
