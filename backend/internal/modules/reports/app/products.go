package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ProductFilters narrow the product reports to a category or a tenant
// attribute's value.
type ProductFilters struct {
	CategoryID                *uuid.UUID
	Attribute, AttributeValue *string
}

// ProductsQuery asks for a page of the products report.
type ProductsQuery struct {
	Range
	ProductFilters
	LocationID    *uuid.UUID
	Sort          string
	Limit, Offset int
}

// ProductsReport returns a page of the products sold in the period, with
// their variants. Costs, profit and margin are zero for an actor who may
// not see costs, who cannot order by profit (ErrCostsHidden).
func (s *Service) ProductsReport(ctx context.Context, a Actor, q ProductsQuery) (_ domain.ProductsReport, err error) {
	ctx, span := s.startSpan(ctx, "ProductsReport", a)
	defer func() { usecase.End(span, err) }()
	defer s.observe(ctx, "products", time.Now())

	f, sort, err := q.filter(a)
	if err != nil {
		return domain.ProductsReport{}, err
	}
	lines, err := s.Sales.SoldVariants(ctx, f)
	if err != nil {
		return domain.ProductsReport{}, err
	}
	r := domain.NewProductsReport(lines, sort, domain.Page{Limit: q.Limit, Offset: q.Offset})
	r.Period, r.Costs = f.Period, true
	if !a.SeesCosts {
		r.HideCosts()
	}
	return r, nil
}

func (q ProductsQuery) filter(a Actor) (domain.ProductFilter, domain.ProductSort, error) {
	period, err := q.period()
	if err != nil {
		return domain.ProductFilter{}, "", err
	}
	sort, err := domain.ParseProductSort(q.Sort)
	if err != nil {
		return domain.ProductFilter{}, "", err
	}
	if sort == domain.ByProfit && !a.SeesCosts {
		return domain.ProductFilter{}, "", domain.ErrCostsHidden
	}
	products, err := q.scope()
	if err != nil {
		return domain.ProductFilter{}, "", err
	}
	return domain.ProductFilter{Scope: scope(a, q.LocationID), ProductScope: products, Period: period}, sort, nil
}

func (f ProductFilters) scope() (domain.ProductScope, error) {
	return domain.NewProductScope(f.CategoryID, f.Attribute, f.AttributeValue)
}

// SaleProfitsQuery asks for a page of the profit by sale report.
type SaleProfitsQuery struct {
	Range
	LocationID    *uuid.UUID
	Limit, Offset int
}

// SaleProfits returns a page of the period's sales with their profit,
// newest first. It needs costs (ErrCostsHidden).
func (s *Service) SaleProfits(ctx context.Context, a Actor, q SaleProfitsQuery) (_ domain.SaleProfitsReport, err error) {
	ctx, span := s.startSpan(ctx, "SaleProfits", a)
	defer func() { usecase.End(span, err) }()
	defer s.observe(ctx, "sale_profits", time.Now())

	if !a.SeesCosts {
		return domain.SaleProfitsReport{}, domain.ErrCostsHidden
	}
	period, err := q.period()
	if err != nil {
		return domain.SaleProfitsReport{}, err
	}
	r, err := s.Sales.SaleProfits(ctx, domain.SaleProfitFilter{
		Scope: scope(a, q.LocationID), Period: period, Page: domain.Page{Limit: q.Limit, Offset: q.Offset},
	})
	if err != nil {
		return domain.SaleProfitsReport{}, err
	}
	r.Period = period
	return r, nil
}
