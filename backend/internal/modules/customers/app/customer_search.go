package app

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ErrInvalidStatus reports a customer search by an unknown status.
var ErrInvalidStatus = errors.New("customers: invalid customer status filter")

// CustomerQuery is a page of a customer search. Status defaults to
// domain.StatusActive; the HTTP adapter bounds Limit and Offset.
type CustomerQuery struct {
	Text      string
	SegmentID *uuid.UUID
	Status    domain.CustomerStatus
	Limit     int
	Offset    int
	// WithBalances asks what each customer of the page owes: the API sets
	// it for whoever sees the receivables.
	WithBalances bool
}

// CustomerPage is a page of customers and how many match the query, with
// what they owe when asked (a customer who owes nothing is not in it).
type CustomerPage struct {
	Items    []*domain.Customer
	Total    int
	Balances map[uuid.UUID]Balance
}

// Balance is what a customer owes on credit and what of it is overdue.
type Balance struct {
	Balance, Overdue decimal.Decimal
}

// CustomerBalances is what the customers owe, from the receivables module
// through a port internal/server adapts.
type CustomerBalances interface {
	Balances(ctx context.Context, tenantID, userID uuid.UUID, customerIDs []uuid.UUID) (map[uuid.UUID]Balance, error)
}

// ListCustomers searches the tenant's customers: the POS's quick search and
// the customers screen.
func (s *Service) ListCustomers(ctx context.Context, a Actor, q CustomerQuery) (_ CustomerPage, err error) {
	ctx, span := s.startSpan(ctx, "ListCustomers", a)
	defer func() { usecase.End(span, err) }()

	status := q.Status
	switch status {
	case "":
		status = domain.StatusActive
	case domain.StatusActive, domain.StatusInactive, domain.StatusAll:
	default:
		return CustomerPage{}, ErrInvalidStatus
	}
	items, total, err := s.Customers.Search(ctx, domain.CustomerFilter{
		TenantID: a.TenantID, Text: strings.TrimSpace(q.Text), SegmentID: q.SegmentID, Status: status,
		Limit: q.Limit, Offset: q.Offset,
	})
	if err != nil {
		return CustomerPage{}, err
	}
	page := CustomerPage{Items: items, Total: total}
	if !q.WithBalances || s.Balances == nil || len(items) == 0 {
		return page, nil
	}
	ids := make([]uuid.UUID, len(items))
	for i, c := range items {
		ids[i] = c.ID()
	}
	if page.Balances, err = s.Balances.Balances(ctx, a.TenantID, a.UserID, ids); err != nil {
		return CustomerPage{}, err
	}
	return page, nil
}
