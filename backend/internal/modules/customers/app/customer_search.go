package app

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

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
}

// CustomerPage is a page of customers and how many match the query.
type CustomerPage struct {
	Items []*domain.Customer
	Total int
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
	return CustomerPage{Items: items, Total: total}, nil
}
