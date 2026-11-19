package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http/reportsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// SalesUseCases are the sales reports' use cases (*app.Service).
type SalesUseCases interface {
	SalesReport(ctx context.Context, a app.Actor, q app.SalesQuery) (domain.SalesReport, error)
	ProductsReport(ctx context.Context, a app.Actor, q app.ProductsQuery) (domain.ProductsReport, error)
	SaleProfits(ctx context.Context, a app.Actor, q app.SaleProfitsQuery) (domain.SaleProfitsReport, error)
}

type salesHandlers struct {
	uc SalesUseCases
	actors
}

// GetSalesReport returns the period's sales by group.
func (h *salesHandlers) GetSalesReport(ctx context.Context, req reportsapi.GetSalesReportRequestObject) (reportsapi.GetSalesReportResponseObject, error) {
	p := req.Params
	r, err := h.uc.SalesReport(ctx, h.actor(ctx), app.SalesQuery{
		Range: days(p.From, p.To), LocationID: p.LocationId, Group: str(p.GroupBy),
	})
	if err != nil {
		return nil, err
	}
	return reportsapi.GetSalesReport200JSONResponse(salesBody(r)), nil
}

func salesBody(r domain.SalesReport) reportsapi.SalesReport {
	items := make([]reportsapi.SalesReportRow, 0, len(r.Rows))
	for _, row := range r.Rows {
		item := reportsapi.SalesReportRow{Key: row.Key, Label: row.Label, SalesCount: row.Sales, Total: money(row.Total)}
		if row.HasBreakdown {
			item.Net, item.Tax = optionalMoney(row.Net, true), optionalMoney(row.Tax, true)
		}
		items = append(items, item)
	}
	t := r.Totals
	return reportsapi.SalesReport{
		Period: period(r.Period), GroupBy: reportsapi.SalesReportGroup(r.Group), Items: items,
		Totals: reportsapi.SalesReportTotals{SalesCount: t.Sales, Net: money(t.Net), Tax: money(t.Tax), Total: money(t.Total)},
	}
}

// GetProductsReport returns a page of the products sold.
func (h *salesHandlers) GetProductsReport(ctx context.Context, req reportsapi.GetProductsReportRequestObject) (reportsapi.GetProductsReportResponseObject, error) {
	p := req.Params
	limit, offset := page(p.Limit, p.Offset)
	r, err := h.uc.ProductsReport(ctx, h.actor(ctx), app.ProductsQuery{
		Range: days(p.From, p.To), LocationID: p.LocationId, Sort: str(p.Sort),
		ProductFilters: app.ProductFilters{CategoryID: p.CategoryId, Attribute: p.Attribute, AttributeValue: p.AttributeValue},
		Limit:          limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	return reportsapi.GetProductsReport200JSONResponse(productsBody(r)), nil
}

func productsBody(r domain.ProductsReport) reportsapi.ProductsReport {
	items := make([]reportsapi.ProductsReportRow, 0, len(r.Rows))
	for _, p := range r.Rows {
		f := profit(p.Profit, r.Costs)
		row := reportsapi.ProductsReportRow{
			ProductId: p.ProductID, Name: p.Name, UnitCode: p.UnitCode, Quantity: quantity(p.Quantity),
			Revenue: f.Revenue, Cost: f.Cost, Profit: f.Profit, Margin: f.Margin,
			Variants: make([]reportsapi.ProductsReportVariant, 0, len(p.Variants)),
		}
		for _, v := range p.Variants {
			vf := profit(v.Profit, r.Costs)
			row.Variants = append(row.Variants, reportsapi.ProductsReportVariant{
				VariantId: v.VariantID, Label: v.Label, Sku: v.SKU, Quantity: quantity(v.Quantity),
				Revenue: vf.Revenue, Cost: vf.Cost, Profit: vf.Profit, Margin: vf.Margin,
			})
		}
		items = append(items, row)
	}
	return reportsapi.ProductsReport{Period: period(r.Period), Items: items, Total: r.Total, Totals: profit(r.Totals, r.Costs)}
}

// GetSaleProfitsReport returns a page of the sales with their profit.
func (h *salesHandlers) GetSaleProfitsReport(ctx context.Context, req reportsapi.GetSaleProfitsReportRequestObject) (reportsapi.GetSaleProfitsReportResponseObject, error) {
	p := req.Params
	limit, offset := page(p.Limit, p.Offset)
	r, err := h.uc.SaleProfits(ctx, h.actor(ctx), app.SaleProfitsQuery{
		Range: days(p.From, p.To), LocationID: p.LocationId, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	return reportsapi.GetSaleProfitsReport200JSONResponse(saleProfitsBody(r)), nil
}

func saleProfitsBody(r domain.SaleProfitsReport) reportsapi.SaleProfitsReport {
	items := make([]reportsapi.SaleProfitRow, 0, len(r.Rows))
	for _, s := range r.Rows {
		f := profit(s.Profit, true)
		items = append(items, reportsapi.SaleProfitRow{
			SaleId: s.SaleID, Number: s.Number, LocationId: s.LocationID, LocationName: s.LocationName,
			UserId: s.UserID, UserName: s.UserName, ConfirmedAt: s.ConfirmedAt,
			Revenue: f.Revenue, Cost: f.Cost, Profit: f.Profit, Margin: f.Margin,
		})
	}
	return reportsapi.SaleProfitsReport{Period: period(r.Period), Items: items, Total: r.Total, Totals: profit(r.Totals, true)}
}
