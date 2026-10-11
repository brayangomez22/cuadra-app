package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LineSnapshot holds a stored line's fields. Gross and Tax are the amounts
// computed when it was sold.
type LineSnapshot struct {
	ProductID   uuid.UUID
	ProductName string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
	TaxRate     decimal.Decimal
	Discount    decimal.Decimal
	Gross       decimal.Decimal
	Tax         decimal.Decimal
}

// PaymentSnapshot holds a stored payment's fields.
type PaymentSnapshot struct {
	Method PaymentMethod
	Amount decimal.Decimal
}

// StoredTotals are the amounts stored with a sale.
type StoredTotals struct {
	Subtotal  decimal.Decimal
	Discounts decimal.Decimal
	Tax       decimal.Decimal
	Total     decimal.Decimal
	Paid      decimal.Decimal
	Change    decimal.Decimal
}

// SaleSnapshot holds a stored sale's fields, for RehydrateSale. Void is set
// only for a voided sale.
type SaleSnapshot struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	LocationID  uuid.UUID
	UserID      uuid.UUID
	CustomerID  *uuid.UUID
	Number      int64
	Status      SaleStatus
	Lines       []LineSnapshot
	Payments    []PaymentSnapshot
	Totals      StoredTotals
	CreatedAt   time.Time
	ConfirmedAt time.Time
	Void        *VoidInfo
}

// Snapshot returns the fields of a stored (confirmed or voided) sale, as the
// repository stores them.
func (s *Sale) Snapshot() SaleSnapshot {
	snap := SaleSnapshot{
		ID: s.id, TenantID: s.tenantID, LocationID: s.locationID, UserID: s.userID,
		CustomerID: copyID(s.customerID), Number: s.number, Status: s.status,
		CreatedAt: s.createdAt, Void: s.VoidInfo(),
	}
	if s.confirmedAt != nil {
		snap.ConfirmedAt = *s.confirmedAt
	}
	for _, l := range s.lines {
		snap.Lines = append(snap.Lines, LineSnapshot{
			ProductID: l.productID, ProductName: l.productName, Quantity: l.quantity,
			UnitPrice: l.unitPrice, TaxRate: l.taxRate, Discount: l.discount, Gross: l.gross, Tax: l.tax,
		})
	}
	for _, p := range s.payments {
		snap.Payments = append(snap.Payments, PaymentSnapshot{Method: p.method, Amount: p.amount})
	}
	t := s.Totals()
	snap.Totals = StoredTotals{
		Subtotal: t.Subtotal, Discounts: t.Discounts, Tax: t.Tax, Total: t.Total, Paid: s.Paid(), Change: s.change,
	}
	return snap
}

// RehydrateSale rebuilds a stored sale. It replays the sale through the same
// rules that built it (lines, payments, number, void) and checks that the
// stored amounts are the ones those rules give: a sale whose lines and
// totals do not add up fails with ErrInconsistentSale instead of becoming an
// entity with wrong amounts. The discount limit is not checked again: it
// applied to whoever sold.
func RehydrateSale(snap SaleSnapshot) (*Sale, error) {
	if snap.Status != SaleConfirmed && snap.Status != SaleVoided {
		return nil, ErrInconsistentSale
	}
	if (snap.Status == SaleVoided) != (snap.Void != nil) || len(snap.Lines) == 0 {
		return nil, ErrInconsistentSale
	}
	s, err := NewSale(NewSaleParams{
		ID: snap.ID, TenantID: snap.TenantID, LocationID: snap.LocationID, UserID: snap.UserID,
		CustomerID: snap.CustomerID, CreatedAt: snap.CreatedAt,
	})
	if err != nil {
		return nil, err
	}
	for _, ls := range snap.Lines {
		l, err := newLine(LineInput{
			ProductID: ls.ProductID, ProductName: ls.ProductName, Quantity: ls.Quantity,
			UnitPrice: ls.UnitPrice, TaxRate: ls.TaxRate, Discount: ls.Discount,
		}, NoDiscountLimit)
		if err != nil {
			return nil, err
		}
		if !l.gross.Equal(ls.Gross) || !l.tax.Equal(ls.Tax) {
			return nil, ErrInconsistentSale
		}
		s.lines = append(s.lines, l)
	}
	payments := make([]Payment, 0, len(snap.Payments))
	for _, ps := range snap.Payments {
		p, err := NewPayment(ps.Method, ps.Amount)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	if err := s.Confirm(payments, snap.ConfirmedAt); err != nil {
		return nil, err
	}
	t, st := s.Totals(), snap.Totals
	if !t.Subtotal.Equal(st.Subtotal) || !t.Discounts.Equal(st.Discounts) || !t.Tax.Equal(st.Tax) ||
		!t.Total.Equal(st.Total) || !s.Paid().Equal(st.Paid) || !s.change.Equal(st.Change) {
		return nil, ErrInconsistentSale
	}
	if err := s.AssignNumber(snap.Number); err != nil {
		return nil, err
	}
	if v := snap.Void; v != nil {
		if err := s.Void(v.UserID, v.Reason, v.At); err != nil {
			return nil, err
		}
	}
	return s, nil
}
