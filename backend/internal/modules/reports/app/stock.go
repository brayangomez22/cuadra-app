package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// IdleQuery asks for a page of the idle stock report.
type IdleQuery struct {
	ProductFilters
	Days          int
	LocationID    *uuid.UUID
	Limit, Offset int
}

// IdleStock returns a page of the variants with stock not sold in the last
// Days days (counted from today in Colombia), by product, the most money
// first (the most quantity for an actor who may not see costs).
func (s *Service) IdleStock(ctx context.Context, a Actor, q IdleQuery) (_ domain.IdleStockReport, err error) {
	ctx, span := s.startSpan(ctx, "IdleStock", a)
	defer func() { usecase.End(span, err) }()
	defer s.observe(ctx, "idle_stock", time.Now())

	since, err := domain.NewSince(s.today(), q.Days)
	if err != nil {
		return domain.IdleStockReport{}, err
	}
	products, err := q.scope()
	if err != nil {
		return domain.IdleStockReport{}, err
	}
	lines, err := s.Stock.IdleVariants(ctx, domain.IdleFilter{Scope: scope(a, q.LocationID), ProductScope: products, Since: since})
	if err != nil {
		return domain.IdleStockReport{}, err
	}
	r := domain.NewIdleStockReport(lines, a.SeesCosts, domain.Page{Limit: q.Limit, Offset: q.Offset})
	r.Since, r.Costs = since, true
	if !a.SeesCosts {
		r.HideCosts()
	}
	return r, nil
}

// AdjustmentsQuery asks for the stock adjustments report.
type AdjustmentsQuery struct {
	Range
	LocationID *uuid.UUID
	Group      string
}

// StockAdjustments returns the period's stock adjustments by user or by
// variant, valued at cost for an actor who sees costs.
func (s *Service) StockAdjustments(ctx context.Context, a Actor, q AdjustmentsQuery) (_ domain.AdjustmentsReport, err error) {
	ctx, span := s.startSpan(ctx, "StockAdjustments", a)
	defer func() { usecase.End(span, err) }()
	defer s.observe(ctx, "stock_adjustments", time.Now())

	period, err := q.period()
	if err != nil {
		return domain.AdjustmentsReport{}, err
	}
	group, err := domain.ParseAdjustmentGroup(q.Group)
	if err != nil {
		return domain.AdjustmentsReport{}, err
	}
	r, err := s.Stock.Adjustments(ctx, domain.AdjustmentFilter{Scope: scope(a, q.LocationID), Period: period, Group: group})
	if err != nil {
		return domain.AdjustmentsReport{}, err
	}
	r.Period, r.Group, r.Costs = period, group, true
	r.Sum()
	if !a.SeesCosts {
		r.HideCosts()
	}
	return r, nil
}
