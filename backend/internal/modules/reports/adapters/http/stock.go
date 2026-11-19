package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http/reportsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// defaultIdleDays is the contract's default for days without selling.
const defaultIdleDays = 60

// StockUseCases are the stock reports' use cases (*app.Service).
type StockUseCases interface {
	IdleStock(ctx context.Context, a app.Actor, q app.IdleQuery) (domain.IdleStockReport, error)
	StockAdjustments(ctx context.Context, a app.Actor, q app.AdjustmentsQuery) (domain.AdjustmentsReport, error)
}

type stockHandlers struct {
	uc StockUseCases
	actors
}

// GetIdleStockReport returns a page of the idle stock.
func (h *stockHandlers) GetIdleStockReport(ctx context.Context, req reportsapi.GetIdleStockReportRequestObject) (reportsapi.GetIdleStockReportResponseObject, error) {
	p := req.Params
	limit, offset := page(p.Limit, p.Offset)
	r, err := h.uc.IdleStock(ctx, h.actor(ctx), app.IdleQuery{
		Days: idleDays(p.Days), LocationID: p.LocationId, Limit: limit, Offset: offset,
		ProductFilters: app.ProductFilters{CategoryID: p.CategoryId, Attribute: p.Attribute, AttributeValue: p.AttributeValue},
	})
	if err != nil {
		return nil, err
	}
	return reportsapi.GetIdleStockReport200JSONResponse(idleBody(r)), nil
}

func idleDays(days *int) int {
	if days == nil {
		return defaultIdleDays
	}
	return *days
}

func idleBody(r domain.IdleStockReport) reportsapi.IdleStockReport {
	items := make([]reportsapi.IdleStockRow, 0, len(r.Rows))
	for _, p := range r.Rows {
		row := reportsapi.IdleStockRow{
			ProductId: p.ProductID, Name: p.Name, UnitCode: p.UnitCode, Quantity: quantity(p.Quantity),
			Value: optionalMoney(p.Value, r.Costs), LastSoldAt: p.LastSoldAt,
			Variants: make([]reportsapi.IdleStockVariant, 0, len(p.Variants)),
		}
		for _, v := range p.Variants {
			row.Variants = append(row.Variants, reportsapi.IdleStockVariant{
				VariantId: v.VariantID, Label: v.Label, Sku: v.SKU, Quantity: quantity(v.Quantity),
				Value: optionalMoney(v.Value, r.Costs), LastSoldAt: v.LastSoldAt,
			})
		}
		items = append(items, row)
	}
	return reportsapi.IdleStockReport{
		Days: r.Since.Days, Since: date(r.Since.Day), Items: items, Total: r.Total, Value: optionalMoney(r.Value, r.Costs),
	}
}

// GetStockAdjustmentsReport returns the period's stock adjustments.
func (h *stockHandlers) GetStockAdjustmentsReport(ctx context.Context, req reportsapi.GetStockAdjustmentsReportRequestObject) (reportsapi.GetStockAdjustmentsReportResponseObject, error) {
	p := req.Params
	q := app.AdjustmentsQuery{Range: days(p.From, p.To), LocationID: p.LocationId, Group: str(p.GroupBy)}
	r, err := h.uc.StockAdjustments(ctx, h.actor(ctx), q)
	if err != nil {
		return nil, err
	}
	return reportsapi.GetStockAdjustmentsReport200JSONResponse(adjustmentsBody(r)), nil
}

func adjustmentsBody(r domain.AdjustmentsReport) reportsapi.StockAdjustmentsReport {
	items := make([]reportsapi.StockAdjustmentsReportRow, 0, len(r.Rows))
	for _, a := range r.Rows {
		row := reportsapi.StockAdjustmentsReportRow{
			Key: a.Key, Label: a.Label, Adjustments: a.Adjustments,
			QuantityIn: quantity(a.QuantityIn), QuantityOut: quantity(a.QuantityOut), Value: optionalMoney(a.Value, r.Costs),
		}
		if a.UnitCode != "" {
			code := a.UnitCode
			row.UnitCode = &code
		}
		items = append(items, row)
	}
	return reportsapi.StockAdjustmentsReport{
		Period: period(r.Period), GroupBy: reportsapi.StockAdjustmentsReportGroup(r.Group), Items: items,
		Adjustments: r.Adjustments, Value: optionalMoney(r.Value, r.Costs),
	}
}
