package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http/reportsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/sheet"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func exportIdleStock(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error) {
	r, err := uc.IdleStock(ctx, a, app.IdleQuery{
		Days: idleDays(p.Days), LocationID: p.LocationId, Limit: sheet.MaxRows,
		ProductFilters: app.ProductFilters{CategoryID: p.CategoryId, Attribute: p.Attribute, AttributeValue: p.AttributeValue},
	})
	if err != nil {
		return sheet.Table{}, err
	}
	t := sheet.Table{Name: "Sin movimiento", Columns: withCosts(r.Costs,
		[]string{"Producto", "Variante", "SKU", "Unidad", "Cantidad", "Última venta"}, "Valor al costo")}
	for _, prod := range r.Rows {
		for _, v := range prod.Variants {
			cells := []sheet.Cell{sheet.Text(prod.Name), sheet.Text(v.Label), sheet.Text(v.SKU), sheet.Text(prod.UnitCode),
				sheet.Number(v.Quantity), lastSold(v.LastSoldAt)}
			if r.Costs {
				cells = append(cells, sheet.Number(domain.Money(v.Value)))
			}
			t.Add(cells...)
		}
	}
	return t, nil
}

func exportCashClosings(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error) {
	period, err := exportRange(p)
	if err != nil {
		return sheet.Table{}, err
	}
	r, err := uc.CashClosings(ctx, a, app.CashQuery{Range: period, LocationID: p.LocationId, Group: str(p.GroupBy)})
	if err != nil {
		return sheet.Table{}, err
	}
	t := sheet.Table{Name: "Cierres de caja", Columns: []string{
		"Grupo", "Cierres", "Con diferencia", "Vendido", "Faltantes", "Sobrantes", "Diferencia neta"}}
	for _, row := range append(r.Rows, domain.CashRow{Label: "Total", CashFigures: r.Totals}) {
		t.Add(sheet.Text(row.Label), sheet.Count(row.Closings), sheet.Count(row.WithDifference),
			sheet.Number(domain.Money(row.SalesTotal)), sheet.Number(domain.Money(row.Shortage)),
			sheet.Number(domain.Money(row.Overage)), sheet.Number(domain.Money(row.Difference)))
	}
	return t, nil
}

func exportTeamControl(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error) {
	period, err := exportRange(p)
	if err != nil {
		return sheet.Table{}, err
	}
	r, err := uc.TeamControl(ctx, a, app.TeamQuery{Range: period, LocationID: p.LocationId, Bucket: str(p.Bucket)})
	if err != nil {
		return sheet.Table{}, err
	}
	t := sheet.Table{Name: "Control del equipo", Columns: withCosts(r.Costs, []string{
		"Usuario", "Anulaciones", "Total anulado", "Descuentos", "Monto descontado", "Descuento promedio",
		"Cambios de precio", "Ajustes", "Cierres con diferencia", "Faltantes", "Sobrantes"}, "Valor de los ajustes")}
	for _, u := range r.Users {
		rateCell := sheet.Text("")
		if dr, ok := u.DiscountRate(); ok {
			rateCell = sheet.Number(dr)
		}
		cells := []sheet.Cell{sheet.Text(u.Name), sheet.Count(u.Voids), sheet.Number(domain.Money(u.VoidsTotal)),
			sheet.Count(u.Discounts), sheet.Number(domain.Money(u.DiscountsAmount)), rateCell, sheet.Count(u.PriceChanges),
			sheet.Count(u.Adjustments), sheet.Count(u.CashDifferences), sheet.Number(domain.Money(u.CashShortage)),
			sheet.Number(domain.Money(u.CashOverage))}
		if r.Costs {
			cells = append(cells, sheet.Number(domain.Money(u.AdjustmentsValue)))
		}
		t.Add(cells...)
	}
	return t, nil
}

func exportAdjustments(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error) {
	period, err := exportRange(p)
	if err != nil {
		return sheet.Table{}, err
	}
	r, err := uc.StockAdjustments(ctx, a, app.AdjustmentsQuery{Range: period, LocationID: p.LocationId, Group: str(p.GroupBy)})
	if err != nil {
		return sheet.Table{}, err
	}
	t := sheet.Table{Name: "Ajustes de inventario", Columns: withCosts(r.Costs,
		[]string{"Grupo", "Unidad", "Ajustes", "Entradas", "Salidas"}, "Valor")}
	for _, row := range r.Rows {
		cells := []sheet.Cell{sheet.Text(row.Label), sheet.Text(row.UnitCode), sheet.Count(row.Adjustments),
			sheet.Number(row.QuantityIn), sheet.Number(row.QuantityOut)}
		if r.Costs {
			cells = append(cells, sheet.Number(domain.Money(row.Value)))
		}
		t.Add(cells...)
	}
	return t, nil
}

func exportReceivables(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error) {
	r, err := uc.ReceivablesAging(ctx, a, str(p.Sort))
	if err != nil {
		return sheet.Table{}, err
	}
	t := sheet.Table{Name: "Cartera", Columns: []string{
		"Cliente", "Por vencer", "1 a 30 días", "31 a 60 días", "61 a 90 días", "Más de 90 días", "Vencido", "Total"}}
	total := r.Totals
	total.Name = "Total"
	for _, l := range append(r.Lines, total) {
		t.Add(sheet.Text(l.Name), sheet.Number(l.Current), sheet.Number(l.Days1To30), sheet.Number(l.Days31To60),
			sheet.Number(l.Days61To90), sheet.Number(l.Over90), sheet.Number(l.Overdue), sheet.Number(l.Total))
	}
	return t, nil
}
