package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SupplierStatus filters a search by whether the suppliers are active.
type SupplierStatus string

// Statuses of a supplier search.
const (
	SuppliersActive   SupplierStatus = "active"
	SuppliersInactive SupplierStatus = "inactive"
	SuppliersAll      SupplierStatus = "all"
)

// SupplierFilter is a page of a supplier search of a tenant. Text matches
// the name (every word, ignoring accents and case) or the NIT's number as a
// prefix.
type SupplierFilter struct {
	TenantID uuid.UUID
	Text     string
	Status   SupplierStatus
	Limit    int
	Offset   int
}

// SupplierRepository stores the suppliers of a tenant. Create and Update
// fail with ErrNITTaken if another supplier of the tenant has the same NIT.
type SupplierRepository interface {
	Create(ctx context.Context, s *Supplier) error
	// Update fails with ErrSupplierNotFound if the supplier is not the
	// tenant's.
	Update(ctx context.Context, s *Supplier) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Supplier, error)
	// Search returns a page of the filter, by name, and how many suppliers
	// match it.
	Search(ctx context.Context, f SupplierFilter) ([]*Supplier, int, error)
}

// PurchaseFilter is a page of the purchases of a tenant, the latest invoice
// first. A nil field does not filter; Status "" means every status. From and
// To bound the invoice date, both included.
type PurchaseFilter struct {
	TenantID   uuid.UUID
	SupplierID *uuid.UUID
	Status     PurchaseStatus
	From       *Date
	To         *Date
	Limit      int
	Offset     int
}

// PurchaseSummary is a purchase as a list shows it, with its supplier's
// name.
type PurchaseSummary struct {
	ID            uuid.UUID
	SupplierID    uuid.UUID
	SupplierName  string
	LocationID    uuid.UUID
	InvoiceNumber string
	IssuedOn      Date
	DueOn         *Date
	Payment       Payment
	Total         decimal.Decimal
	BalanceDue    decimal.Decimal
	Lines         int
	Voided        bool
	CreatedAt     time.Time
}

// Status returns voided once voided, open while something is left to pay,
// and paid otherwise.
func (s PurchaseSummary) Status() PurchaseStatus {
	if s.Voided {
		return PurchaseVoided
	}
	if s.BalanceDue.IsPositive() {
		return PurchaseOpen
	}
	return PurchasePaid
}

// PurchaseRepository stores the purchases of a tenant.
type PurchaseRepository interface {
	// Save stores a new purchase and calls receive in the same transaction
	// (through ctx): after checking that the supplier's invoice is new
	// (ErrInvoiceTaken) and before storing the lines, so they keep what
	// receive recorded on p. If receive fails, nothing is stored.
	Save(ctx context.Context, p *Purchase, receive func(context.Context) error) error
	// GetByID fails with ErrPurchaseNotFound if the purchase is not the
	// tenant's.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Purchase, error)
	// Search returns a page of the filter and how many purchases match it.
	Search(ctx context.Context, f PurchaseFilter) ([]PurchaseSummary, int, error)
	// BalancesDue returns what the tenant owes to each supplier among ids
	// that it owes something: the sum of its purchases' balances.
	BalancesDue(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]decimal.Decimal, error)
	// Void locks the tenant's purchase (ErrPurchaseNotFound), calls void
	// with it in the same transaction (through ctx) and stores it voided.
	// If void fails, nothing changes.
	Void(ctx context.Context, tenantID, id uuid.UUID, void func(context.Context, *Purchase) error) (*Purchase, error)
	// Pay locks the tenant's purchase (ErrPurchaseNotFound), calls pay with
	// it and stores its new balance and the payment pay returns, together.
	Pay(ctx context.Context, tenantID, id uuid.UUID, pay func(*Purchase) (*SupplierPayment, error)) (*Purchase, *SupplierPayment, error)
	// Payments returns the payments of a purchase of the tenant, the first
	// one first.
	Payments(ctx context.Context, tenantID, purchaseID uuid.UUID) ([]*SupplierPayment, error)
	// Payables returns a page of what the tenant owes each supplier it owes
	// something, the one with the most overdue first, split by w, and the
	// totals of every supplier.
	Payables(ctx context.Context, tenantID uuid.UUID, w PayablesWindow, limit, offset int) (PayablesPage, error)
}

// PayablesPage is a page of what the tenant owes each supplier, the totals
// of all of them and how many suppliers it owes.
type PayablesPage struct {
	Items  []SupplierPayable
	Totals PayablesTotals
}
