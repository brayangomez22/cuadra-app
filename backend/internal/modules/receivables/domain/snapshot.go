package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountSnapshot holds a stored account's fields, for RehydrateAccount.
type AccountSnapshot struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	Limit      decimal.Decimal
	TermDays   int
	Balance    decimal.Decimal
}

// RehydrateAccount rebuilds a stored account. It trusts the snapshot: the
// database's constraints already hold the invariants.
func RehydrateAccount(s AccountSnapshot) *Account {
	return &Account{
		tenantID: s.TenantID, customerID: s.CustomerID,
		terms: Terms{limit: s.Limit, days: s.TermDays}, balance: s.Balance,
	}
}

// ReceivableSnapshot holds a stored receivable's fields.
type ReceivableSnapshot struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	CustomerID    uuid.UUID
	SaleID        uuid.UUID
	Amount        decimal.Decimal
	Balance       decimal.Decimal
	DueDate       Date
	Status        ReceivableStatus
	Authorization *Authorization
	CreatedAt     time.Time
	CanceledAt    *time.Time
}

// RehydrateReceivable rebuilds a stored receivable.
func RehydrateReceivable(s ReceivableSnapshot) *Receivable {
	r := &Receivable{
		id: s.ID, tenantID: s.TenantID, customerID: s.CustomerID, saleID: s.SaleID,
		amount: s.Amount, balance: s.Balance, dueDate: s.DueDate, status: s.Status,
		createdAt: s.CreatedAt, canceledAt: s.CanceledAt,
	}
	if s.Authorization != nil {
		a := *s.Authorization
		r.authorization = &a
	}
	return r
}

// Snapshot returns the receivable's fields, for storage.
func (r *Receivable) Snapshot() ReceivableSnapshot {
	return ReceivableSnapshot{
		ID: r.id, TenantID: r.tenantID, CustomerID: r.customerID, SaleID: r.saleID,
		Amount: r.amount, Balance: r.balance, DueDate: r.dueDate, Status: r.status,
		Authorization: r.Authorization(), CreatedAt: r.createdAt, CanceledAt: r.CanceledAt(),
	}
}
