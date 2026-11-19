package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Balance is what a customer owes and what of it is overdue.
type Balance struct {
	Balance, Overdue decimal.Decimal
}

// Balances returns what each of the customers owes, as of today in
// America/Bogota; a customer who owes nothing is not in the map. It is for
// the customers module's list, through a port internal/server adapts.
func (s *Service) Balances(ctx context.Context, a Actor, customerIDs []uuid.UUID) (_ map[uuid.UUID]Balance, err error) {
	ctx, span := s.startSpan(ctx, "Balances", a)
	defer func() { usecase.End(span, err) }()

	open, err := s.Credit.OpenBalances(ctx, a.TenantID)
	if err != nil {
		return nil, err
	}
	asked := make(map[uuid.UUID]bool, len(customerIDs))
	for _, id := range customerIDs {
		asked[id] = true
	}
	today := domain.DateIn(s.Now())
	byCustomer := map[uuid.UUID]*domain.Aging{}
	for _, b := range open {
		if !asked[b.CustomerID] {
			continue
		}
		if byCustomer[b.CustomerID] == nil {
			byCustomer[b.CustomerID] = domain.NewAging(today)
		}
		byCustomer[b.CustomerID].Add(b.DueDate, b.Balance)
	}
	out := make(map[uuid.UUID]Balance, len(byCustomer))
	for id, g := range byCustomer {
		out[id] = Balance{Balance: g.Total(), Overdue: g.Overdue()}
	}
	return out, nil
}
