package app

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// StatementPage is a customer's account and a page of its statement.
type StatementPage struct {
	Account   *domain.Account
	Movements []domain.Movement
	Total     int
}

// AgingRow is what a customer owes, by bucket.
type AgingRow struct {
	CustomerID uuid.UUID
	Name       string
	Aging      *domain.Aging
}

// AgingReport is a page of the aging by customer, as of a day, with the
// totals of all of it.
type AgingReport struct {
	AsOf   domain.Date
	Rows   []AgingRow
	Totals *domain.Aging
	Total  int
}

// Statement returns a customer's account and a page of its statement:
// credit sales, payments and canceled sales, newest first, each with the
// balance after it.
func (s *Service) Statement(ctx context.Context, a Actor, customerID uuid.UUID, limit, offset int) (_ StatementPage, err error) {
	ctx, span := s.startSpan(ctx, "Statement", a, customerAttr(customerID))
	defer func() { usecase.End(span, err) }()

	if err := s.Customers.CheckCustomer(ctx, a.TenantID, a.UserID, customerID); err != nil {
		return StatementPage{}, err
	}
	account, err := s.Credit.Account(ctx, a.TenantID, customerID)
	if err != nil {
		return StatementPage{}, err
	}
	movements, err := s.Credit.Movements(ctx, a.TenantID, customerID)
	if err != nil {
		return StatementPage{}, err
	}
	all := domain.Statement(movements)
	page := all[min(offset, len(all)):min(offset+limit, len(all))]
	return StatementPage{Account: account, Movements: page, Total: len(all)}, nil
}

// Aging returns a page of what the tenant's customers owe by bucket, as of
// today in America/Bogota: the customers with the most overdue first (then
// the most owed), with the totals of every customer.
func (s *Service) Aging(ctx context.Context, a Actor, limit, offset int) (_ AgingReport, err error) {
	ctx, span := s.startSpan(ctx, "Aging", a)
	defer func() { usecase.End(span, err) }()

	balances, err := s.Credit.OpenBalances(ctx, a.TenantID)
	if err != nil {
		return AgingReport{}, err
	}
	report := AgingReport{AsOf: domain.DateIn(s.Now())}
	report.Totals = domain.NewAging(report.AsOf)
	rows := agingRows(report.AsOf, balances, report.Totals)
	report.Total = len(rows)
	page := rows[min(offset, len(rows)):min(offset+limit, len(rows))]
	if len(page) == 0 {
		return report, nil
	}
	ids := make([]uuid.UUID, len(page))
	for i, r := range page {
		ids[i] = r.CustomerID
	}
	names, err := s.Customers.Names(ctx, a.TenantID, a.UserID, ids)
	if err != nil {
		return AgingReport{}, err
	}
	for i := range page {
		page[i].Name = names[page[i].CustomerID]
	}
	report.Rows = page
	return report, nil
}

// agingRows adds up the balances by customer and into totals, and orders
// the customers by what they owe overdue, then by what they owe, then by id.
func agingRows(asOf domain.Date, balances []domain.OpenBalance, totals *domain.Aging) []AgingRow {
	byCustomer := map[uuid.UUID]*domain.Aging{}
	for _, b := range balances {
		g, ok := byCustomer[b.CustomerID]
		if !ok {
			g = domain.NewAging(asOf)
			byCustomer[b.CustomerID] = g
		}
		g.Add(b.DueDate, b.Balance)
		totals.Add(b.DueDate, b.Balance)
	}
	rows := make([]AgingRow, 0, len(byCustomer))
	for id, g := range byCustomer {
		rows = append(rows, AgingRow{CustomerID: id, Aging: g})
	}
	slices.SortFunc(rows, func(x, y AgingRow) int {
		if c := y.Aging.Overdue().Cmp(x.Aging.Overdue()); c != 0 {
			return c
		}
		if c := y.Aging.Total().Cmp(x.Aging.Total()); c != 0 {
			return c
		}
		return strings.Compare(x.CustomerID.String(), y.CustomerID.String())
	})
	return rows
}
