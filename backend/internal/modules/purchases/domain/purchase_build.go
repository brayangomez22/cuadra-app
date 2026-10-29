package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// header is a purchase's data other than its lines, as registered or
// stored.
type header struct {
	id, tenantID, supplierID, locationID uuid.UUID
	invoiceNumber                        string
	issuedOn                             Date
	dueOn                                *Date
	payment                              Payment
	notes                                string
	createdBy                            uuid.UUID
	createdAt                            time.Time
}

// buildPurchase checks h and the lines, the invariants of every purchase
// (registered now or stored), and computes the totals. The balance is zero.
func buildPurchase(h header, lines []LineInput) (*Purchase, error) {
	if err := h.checkIDs(); err != nil {
		return nil, err
	}
	number, key, err := parseInvoiceNumber(h.invoiceNumber)
	if err != nil {
		return nil, err
	}
	notes, err := optionalText(h.notes, maxNotesLength, ErrInvalidNotes)
	if err != nil {
		return nil, err
	}
	if err := h.checkPayment(); err != nil {
		return nil, err
	}
	p := &Purchase{
		id: h.id, tenantID: h.tenantID, supplierID: h.supplierID, locationID: h.locationID,
		invoiceNumber: number, invoiceKey: key, issuedOn: h.issuedOn, dueOn: h.dueOn, payment: h.payment,
		notes: notes, balanceDue: decimal.Zero, createdBy: h.createdBy, createdAt: h.createdAt.UTC(),
	}
	if p.lines, err = buildLines(lines); err != nil {
		return nil, err
	}
	p.totals = computeTotals(p.lines)
	return p, nil
}

func (h header) checkIDs() error {
	switch uuid.Nil {
	case h.id:
		return ErrInvalidPurchaseID
	case h.tenantID:
		return ErrInvalidTenantID
	case h.supplierID:
		return ErrInvalidSupplierID
	case h.locationID:
		return ErrInvalidLocationID
	case h.createdBy:
		return ErrInvalidUserID
	}
	return nil
}

// checkPayment checks the payment and its due date: only a purchase on
// credit has one, never before the invoice date.
func (h header) checkPayment() error {
	switch h.payment {
	case PaymentCash:
		if h.dueOn != nil {
			return ErrInvalidPayment
		}
	case PaymentCredit:
		if h.dueOn == nil || h.dueOn.Before(h.issuedOn) {
			return ErrDueBeforeIssue
		}
	default:
		return ErrInvalidPayment
	}
	return nil
}

func buildLines(in []LineInput) ([]Line, error) {
	if len(in) == 0 {
		return nil, ErrNoLines
	}
	if len(in) > maxPurchaseLines {
		return nil, ErrTooManyLines
	}
	lines := make([]Line, len(in))
	for i, l := range in {
		var err error
		if lines[i], err = newLine(l); err != nil {
			return nil, err
		}
	}
	return lines, nil
}
