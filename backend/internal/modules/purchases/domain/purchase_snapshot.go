package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LineSnapshot holds a purchase line's fields as they are stored.
type LineSnapshot struct {
	LineInput
	Received
}

// PurchaseSnapshot holds a purchase's fields as they are stored. The totals
// are not stored apart from the lines: they are recomputed.
type PurchaseSnapshot struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	SupplierID    uuid.UUID
	LocationID    uuid.UUID
	InvoiceNumber string
	IssuedOn      Date
	DueOn         *Date
	Payment       Payment
	Notes         string
	Lines         []LineSnapshot
	BalanceDue    decimal.Decimal
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
}

// Snapshot returns the purchase's fields, for the repository.
func (p *Purchase) Snapshot() PurchaseSnapshot {
	s := PurchaseSnapshot{
		ID: p.id, TenantID: p.tenantID, SupplierID: p.supplierID, LocationID: p.locationID,
		InvoiceNumber: p.invoiceNumber, IssuedOn: p.issuedOn, DueOn: p.DueOn(), Payment: p.payment,
		Notes: p.notes, BalanceDue: p.balanceDue, CreatedBy: p.createdBy, CreatedAt: p.createdAt,
	}
	for _, l := range p.Lines() {
		s.Lines = append(s.Lines, LineSnapshot{LineInput: l.in, Received: l.received})
	}
	return s
}

// RehydratePurchase rebuilds a stored purchase, checking the invariants of
// every purchase: its lines were received and its balance is between zero
// and its total (zero in cash). The rules of registering (an active
// supplier, the invoice date) held when it was registered.
func RehydratePurchase(s PurchaseSnapshot) (*Purchase, error) {
	lines := make([]LineInput, len(s.Lines))
	received := make([]Received, len(s.Lines))
	for i, l := range s.Lines {
		lines[i], received[i] = l.LineInput, l.Received
	}
	p, err := buildPurchase(header{
		id: s.ID, tenantID: s.TenantID, supplierID: s.SupplierID, locationID: s.LocationID,
		invoiceNumber: s.InvoiceNumber, issuedOn: s.IssuedOn, dueOn: s.DueOn, payment: s.Payment,
		notes: s.Notes, createdBy: s.CreatedBy, createdAt: s.CreatedAt,
	}, lines)
	if err != nil {
		return nil, err
	}
	if err := p.Receive(received); err != nil {
		return nil, ErrInvalidPurchase
	}
	if s.BalanceDue.IsNegative() || s.BalanceDue.GreaterThan(p.totals.Total) ||
		(s.Payment == PaymentCash && !s.BalanceDue.IsZero()) {
		return nil, ErrInvalidPurchase
	}
	p.balanceDue = s.BalanceDue
	return p, nil
}
