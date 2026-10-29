package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	// maxPurchaseLines bounds a purchase so it is stored in one transaction
	// of reasonable size; a longer invoice is registered in parts.
	maxPurchaseLines = 200
	maxNotesLength   = 500
)

// Payment is how a purchase is paid: in cash (paid when registered) or on
// credit (it leaves a balance to pay to the supplier).
type Payment string

// Payments of a purchase.
const (
	PaymentCash   Payment = "cash"
	PaymentCredit Payment = "credit"
)

// PurchaseStatus tells whether a purchase still owes something.
type PurchaseStatus string

// Statuses of a purchase.
const (
	PurchaseOpen PurchaseStatus = "open"
	PurchasePaid PurchaseStatus = "paid"
)

// PurchaseParams are the data of a supplier's invoice. DueOn is optional:
// without it, a purchase on credit falls due after the supplier's term; it
// is ignored in cash.
type PurchaseParams struct {
	Supplier      *Supplier
	LocationID    uuid.UUID
	InvoiceNumber string
	IssuedOn      Date
	DueOn         *Date
	Payment       Payment
	Notes         string
	CreatedBy     uuid.UUID
}

// Purchase is a supplier's invoice registered by the tenant: the goods that
// entered a location, line by line in the unit they were bought in, and
// what is left to pay. Its lines enter the stock when it is registered
// (Receive), in the same transaction.
type Purchase struct {
	id            uuid.UUID
	tenantID      uuid.UUID
	supplierID    uuid.UUID
	locationID    uuid.UUID
	invoiceNumber string
	invoiceKey    string
	issuedOn      Date
	dueOn         *Date
	payment       Payment
	notes         string
	lines         []Line
	totals        Totals
	balanceDue    decimal.Decimal
	createdBy     uuid.UUID
	createdAt     time.Time
}

// NewPurchase registers a supplier's invoice with its own UUID v7. The
// supplier must be active and the invoice date not after today in
// America/Bogota. A purchase on credit owes its total.
func NewPurchase(tenantID uuid.UUID, p PurchaseParams, lines []LineInput, now time.Time) (*Purchase, error) {
	if err := checkRegistration(tenantID, p, now); err != nil {
		return nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	pu, err := buildPurchase(header{
		id: id, tenantID: tenantID, supplierID: p.Supplier.ID(), locationID: p.LocationID,
		invoiceNumber: p.InvoiceNumber, issuedOn: p.IssuedOn, dueOn: p.dueOn(), payment: p.Payment, notes: p.Notes,
		createdBy: p.CreatedBy, createdAt: now,
	}, lines)
	if err != nil {
		return nil, err
	}
	if pu.payment == PaymentCredit {
		pu.balanceDue = pu.totals.Total
	}
	return pu, nil
}

// checkRegistration checks the rules of registering a purchase now: the
// tenant's active supplier and an invoice date not after today.
func checkRegistration(tenantID uuid.UUID, p PurchaseParams, now time.Time) error {
	if p.Supplier == nil || p.Supplier.TenantID() != tenantID {
		return ErrSupplierNotFound
	}
	if !p.Supplier.Active() {
		return ErrSupplierInactive
	}
	if p.IssuedOn.IsZero() || DateIn(now).Before(p.IssuedOn) {
		return ErrInvoiceDateInFuture
	}
	return nil
}

// dueOn is when a purchase on credit falls due: DueOn or, without it, the
// supplier's term after the invoice. A purchase in cash has none.
func (p PurchaseParams) dueOn() *Date {
	if p.Payment != PaymentCredit {
		return nil
	}
	due := p.Supplier.dueAfter(p.IssuedOn)
	if p.DueOn != nil {
		due = *p.DueOn
	}
	return &due
}

// Receive records what each line put into the stock, in line order.
func (p *Purchase) Receive(received []Received) error {
	if len(received) != len(p.lines) {
		return ErrReceiptMismatch
	}
	for _, r := range received {
		if !r.valid() {
			return ErrReceiptMismatch
		}
	}
	for i, r := range received {
		p.lines[i].received = r
	}
	return nil
}

// ID returns the purchase ID (UUID v7).
func (p *Purchase) ID() uuid.UUID { return p.id }

// TenantID returns the tenant the purchase belongs to.
func (p *Purchase) TenantID() uuid.UUID { return p.tenantID }

// SupplierID returns the supplier that issued the invoice.
func (p *Purchase) SupplierID() uuid.UUID { return p.supplierID }

// LocationID returns the location the goods entered.
func (p *Purchase) LocationID() uuid.UUID { return p.locationID }

// InvoiceNumber returns the supplier's invoice number, as written.
func (p *Purchase) InvoiceNumber() string { return p.invoiceNumber }

// InvoiceKey returns the invoice number without spaces, dashes or dots and
// in capitals: unique per supplier.
func (p *Purchase) InvoiceKey() string { return p.invoiceKey }

// IssuedOn returns the invoice date.
func (p *Purchase) IssuedOn() Date { return p.issuedOn }

// DueOn returns a copy of the due date of a purchase on credit, or nil in
// cash.
func (p *Purchase) DueOn() *Date {
	if p.dueOn == nil {
		return nil
	}
	d := *p.dueOn
	return &d
}

// Payment returns whether the purchase was in cash or on credit.
func (p *Purchase) Payment() Payment { return p.payment }

// Notes returns the free notes of the purchase.
func (p *Purchase) Notes() string { return p.notes }

// Lines returns a copy of the lines, in the invoice's order.
func (p *Purchase) Lines() []Line {
	out := make([]Line, len(p.lines))
	for i, l := range p.lines {
		l.in.Unit = l.in.Unit.clone()
		out[i] = l
	}
	return out
}

// Totals returns the purchase's amounts.
func (p *Purchase) Totals() Totals {
	t := p.totals
	t.Taxes = append([]TaxLine(nil), p.totals.Taxes...)
	return t
}

// BalanceDue returns what is left to pay to the supplier.
func (p *Purchase) BalanceDue() decimal.Decimal { return p.balanceDue }

// Status returns open while something is left to pay, paid otherwise.
func (p *Purchase) Status() PurchaseStatus {
	if p.balanceDue.IsPositive() {
		return PurchaseOpen
	}
	return PurchasePaid
}

// CreatedBy returns the user who registered the purchase.
func (p *Purchase) CreatedBy() uuid.UUID { return p.createdBy }

// CreatedAt returns when it was registered, in UTC.
func (p *Purchase) CreatedAt() time.Time { return p.createdAt }
