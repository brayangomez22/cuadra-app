package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SaleStatus is the state of a sale.
type SaleStatus string

// Sale states. A sale is built as a draft and confirmed once paid; voiding
// comes with persistence (T20).
const (
	SaleDraft     SaleStatus = "draft"
	SaleConfirmed SaleStatus = "confirmed"
)

func (s SaleStatus) String() string { return string(s) }

// NewSaleParams are the fields of a new sale. CustomerID is optional.
type NewSaleParams struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	LocationID uuid.UUID
	// UserID is who sells: the cashier.
	UserID     uuid.UUID
	CustomerID *uuid.UUID
	CreatedAt  time.Time
}

// TaxBreakdown is the IVA of the lines at one rate: the sum of their bases
// and of their taxes.
type TaxBreakdown struct {
	Rate decimal.Decimal
	Base decimal.Decimal
	Tax  decimal.Decimal
}

// Totals are a sale's amounts. Subtotal is the sum of the lines' gross
// amounts; Total = Subtotal − Discounts + Tax. Taxes has one entry per IVA
// rate, by rate.
type Totals struct {
	Subtotal  decimal.Decimal
	Discounts decimal.Decimal
	Tax       decimal.Decimal
	Total     decimal.Decimal
	Taxes     []TaxBreakdown
}

// Sale is a point-of-sale sale at a location: its lines and, once
// confirmed, its payments. Lines are added only to a draft; a confirmed sale
// does not change.
type Sale struct {
	id          uuid.UUID
	tenantID    uuid.UUID
	locationID  uuid.UUID
	userID      uuid.UUID
	customerID  *uuid.UUID
	status      SaleStatus
	lines       []SaleLine
	payments    []Payment
	change      decimal.Decimal
	createdAt   time.Time
	confirmedAt *time.Time
}

// NewSale creates an empty draft sale.
func NewSale(p NewSaleParams) (*Sale, error) {
	if p.ID == uuid.Nil {
		return nil, ErrInvalidSaleID
	}
	if p.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if p.LocationID == uuid.Nil {
		return nil, ErrInvalidLocationID
	}
	if p.UserID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if p.CustomerID != nil && *p.CustomerID == uuid.Nil {
		return nil, ErrInvalidCustomerID
	}
	return &Sale{
		id:         p.ID,
		tenantID:   p.TenantID,
		locationID: p.LocationID,
		userID:     p.UserID,
		customerID: copyID(p.CustomerID),
		status:     SaleDraft,
		createdAt:  p.CreatedAt.UTC(),
	}, nil
}

func copyID(id *uuid.UUID) *uuid.UUID {
	if id == nil {
		return nil
	}
	c := *id
	return &c
}

// AddLine adds a line to a draft sale. Its discount must be within limit,
// the limit of the user who sells. The same product may appear in several
// lines (with different discounts, for instance).
func (s *Sale) AddLine(in LineInput, limit DiscountLimit) error {
	if s.status != SaleDraft {
		return ErrSaleNotDraft
	}
	l, err := newLine(in, limit)
	if err != nil {
		return err
	}
	s.lines = append(s.lines, l)
	return nil
}

// Confirm pays a draft sale with payments and confirms it at the given time.
// The payments must cover the total; only cash gives change, so the other
// methods together cannot exceed the total, and the change (paid − total) is
// never more than the cash received. On error the sale is unchanged.
func (s *Sale) Confirm(payments []Payment, at time.Time) error {
	if s.status != SaleDraft {
		return ErrSaleNotDraft
	}
	if len(s.lines) == 0 {
		return ErrSaleWithoutLines
	}
	total := s.Totals().Total
	paid, nonCash := decimal.Zero, decimal.Zero
	for _, p := range payments {
		paid = paid.Add(p.amount)
		if p.method != PaymentCash {
			nonCash = nonCash.Add(p.amount)
		}
	}
	if paid.LessThan(total) {
		return ErrInsufficientPayment
	}
	if nonCash.GreaterThan(total) {
		return ErrNonCashOverpayment
	}
	confirmedAt := at.UTC()
	s.payments = slices.Clone(payments)
	s.change = paid.Sub(total)
	s.status = SaleConfirmed
	s.confirmedAt = &confirmedAt
	return nil
}

// Totals adds up the lines. The taxes by rate are the sums of the lines'
// already rounded amounts, so nothing is rounded here.
func (s *Sale) Totals() Totals {
	t := Totals{Subtotal: decimal.Zero, Discounts: decimal.Zero, Tax: decimal.Zero}
	byRate := map[string]int{}
	for _, l := range s.lines {
		t.Subtotal = t.Subtotal.Add(l.gross)
		t.Discounts = t.Discounts.Add(l.discount)
		t.Tax = t.Tax.Add(l.tax)

		// The key normalizes 0.19 and 0.1900 to the same rate.
		key := l.taxRate.String()
		i, ok := byRate[key]
		if !ok {
			i = len(t.Taxes)
			byRate[key] = i
			t.Taxes = append(t.Taxes, TaxBreakdown{Rate: l.taxRate, Base: decimal.Zero, Tax: decimal.Zero})
		}
		t.Taxes[i].Base = t.Taxes[i].Base.Add(l.Base())
		t.Taxes[i].Tax = t.Taxes[i].Tax.Add(l.tax)
	}
	slices.SortFunc(t.Taxes, func(a, b TaxBreakdown) int { return a.Rate.Cmp(b.Rate) })
	t.Total = t.Subtotal.Sub(t.Discounts).Add(t.Tax)
	return t
}

// ID returns the sale ID (UUID v7).
func (s *Sale) ID() uuid.UUID { return s.id }

// TenantID returns the tenant the sale belongs to.
func (s *Sale) TenantID() uuid.UUID { return s.tenantID }

// LocationID returns where the sale happened and the stock comes from.
func (s *Sale) LocationID() uuid.UUID { return s.locationID }

// UserID returns who sold.
func (s *Sale) UserID() uuid.UUID { return s.userID }

// CustomerID returns a copy of the customer's id, or nil for an anonymous sale.
func (s *Sale) CustomerID() *uuid.UUID { return copyID(s.customerID) }

// Status returns the sale's state.
func (s *Sale) Status() SaleStatus { return s.status }

// Lines returns a copy of the lines, in the order they were added.
func (s *Sale) Lines() []SaleLine { return slices.Clone(s.lines) }

// Payments returns a copy of the payments; empty for a draft.
func (s *Sale) Payments() []Payment { return slices.Clone(s.payments) }

// Paid returns the sum of the payments.
func (s *Sale) Paid() decimal.Decimal {
	paid := decimal.Zero
	for _, p := range s.payments {
		paid = paid.Add(p.amount)
	}
	return paid
}

// Change returns the cash given back; zero for a draft.
func (s *Sale) Change() decimal.Decimal { return s.change }

// CreatedAt returns when the sale was started, in UTC.
func (s *Sale) CreatedAt() time.Time { return s.createdAt }

// ConfirmedAt returns a copy of when the sale was confirmed, in UTC, or nil
// for a draft.
func (s *Sale) ConfirmedAt() *time.Time {
	if s.confirmedAt == nil {
		return nil
	}
	c := *s.confirmedAt
	return &c
}
