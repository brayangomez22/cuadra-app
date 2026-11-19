package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http/reportsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/sheet"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// methodNames are the payment methods as the business says them.
var methodNames = map[string]string{
	"cash": "Efectivo", "card": "Tarjeta", "nequi": "Nequi", "daviplata": "Daviplata",
	"transfer": "Transferencia", "credit": "Crédito",
}

func exportSales(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error) {
	period, err := exportRange(p)
	if err != nil {
		return sheet.Table{}, err
	}
	r, err := uc.SalesReport(ctx, a, app.SalesQuery{Range: period, LocationID: p.LocationId, Group: str(p.GroupBy)})
	if err != nil {
		return sheet.Table{}, err
	}
	t := sheet.Table{Name: "Ventas", Columns: []string{"Grupo", "Ventas", "Neto sin IVA", "IVA", "Total"}}
	for _, row := range r.Rows {
		label := row.Label
		if r.Group == domain.SalesByPaymentMethod {
			label = methodNames[row.Key]
		}
		t.Add(sheet.Text(label), sheet.Count(row.Sales), figure(row.Net, row.HasBreakdown), figure(row.Tax, row.HasBreakdown),
			sheet.Number(domain.Money(row.Total)))
	}
	t.Add(sheet.Text("Total"), sheet.Count(r.Totals.Sales), sheet.Number(domain.Money(r.Totals.Net)),
		sheet.Number(domain.Money(r.Totals.Tax)), sheet.Number(domain.Money(r.Totals.Total)))
	return t, nil
}

func exportProducts(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error) {
	period, err := exportRange(p)
	if err != nil {
		return sheet.Table{}, err
	}
	r, err := uc.ProductsReport(ctx, a, app.ProductsQuery{
		Range: period, LocationID: p.LocationId, Sort: str(p.Sort), Limit: sheet.MaxRows,
		ProductFilters: app.ProductFilters{CategoryID: p.CategoryId, Attribute: p.Attribute, AttributeValue: p.AttributeValue},
	})
	if err != nil {
		return sheet.Table{}, err
	}
	t := sheet.Table{Name: "Productos", Columns: withCosts(r.Costs,
		[]string{"Producto", "Variante", "SKU", "Unidad", "Cantidad", "Ingreso sin IVA"}, "Costo", "Utilidad", "Margen")}
	for _, prod := range r.Rows {
		for _, v := range prod.Variants {
			cells := []sheet.Cell{sheet.Text(prod.Name), sheet.Text(v.Label), sheet.Text(v.SKU), sheet.Text(prod.UnitCode),
				sheet.Number(v.Quantity), sheet.Number(domain.Money(v.Revenue))}
			t.Add(append(cells, profitCells(v.Profit, r.Costs)...)...)
		}
	}
	return t, nil
}

func exportSaleProfits(ctx context.Context, uc UseCases, a app.Actor, p reportsapi.ExportReportParams) (sheet.Table, error) {
	period, err := exportRange(p)
	if err != nil {
		return sheet.Table{}, err
	}
	r, err := uc.SaleProfits(ctx, a, app.SaleProfitsQuery{Range: period, LocationID: p.LocationId, Limit: sheet.MaxRows})
	if err != nil {
		return sheet.Table{}, err
	}
	t := sheet.Table{Name: "Utilidad por venta", Columns: []string{
		"Fecha", "Sede", "Número", "Vendedor", "Ingreso sin IVA", "Costo", "Utilidad", "Margen"}}
	for _, s := range r.Rows {
		cells := []sheet.Cell{sheet.Text(s.ConfirmedAt.In(bogota).Format("2006-01-02 15:04")), sheet.Text(s.LocationName),
			sheet.Count(int(s.Number)), sheet.Text(s.UserName), sheet.Number(domain.Money(s.Revenue))}
		t.Add(append(cells, profitCells(s.Profit, true)...)...)
	}
	return t, nil
}
