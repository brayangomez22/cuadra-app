package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// SalesQuery asks for the sales report; Until (HH:MM) ends its last day at
// that time.
type SalesQuery struct {
	Range
	LocationID *uuid.UUID
	Group      string
	Until      string
}

// SalesReport returns the period's confirmed sales by group. The
// permission is checked by the API (reports:read).
func (s *Service) SalesReport(ctx context.Context, a Actor, q SalesQuery) (_ domain.SalesReport, err error) {
	ctx, span := s.startSpan(ctx, "SalesReport", a)
	defer func() { usecase.End(span, err) }()
	defer s.observe(ctx, "sales", time.Now())

	period, err := q.period()
	if err == nil {
		period, err = period.Until(q.Until)
	}
	if err != nil {
		return domain.SalesReport{}, err
	}
	group, err := domain.ParseSalesGroup(q.Group)
	if err != nil {
		return domain.SalesReport{}, err
	}
	r, err := s.Sales.Sales(ctx, domain.SalesFilter{Scope: scope(a, q.LocationID), Period: period, Group: group})
	if err != nil {
		return domain.SalesReport{}, err
	}
	r.Period, r.Group = period, group
	return r, nil
}
