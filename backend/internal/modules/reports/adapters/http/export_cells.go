package httpadapter

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/sheet"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// bogota is where the exported dates and times are read.
var bogota = time.FixedZone("America/Bogota", -5*3600)

// withCosts adds the cost columns to columns for whoever sees costs.
func withCosts(costs bool, columns []string, costColumns ...string) []string {
	if !costs {
		return columns
	}
	return append(columns, costColumns...)
}

// figure is an amount in pesos, or an empty cell when the row has none.
func figure(d decimal.Decimal, ok bool) sheet.Cell {
	if !ok {
		return sheet.Text("")
	}
	return sheet.Number(domain.Money(d))
}

// profitCells are the cost, profit and margin of p, for whoever sees costs.
func profitCells(p domain.Profit, costs bool) []sheet.Cell {
	if !costs {
		return nil
	}
	m, ok := p.Margin()
	margin := sheet.Text("")
	if ok {
		margin = sheet.Number(m)
	}
	return []sheet.Cell{sheet.Number(domain.Money(p.Cost)), sheet.Number(p.Amount()), margin}
}

// lastSold is when a variant last sold, or "Nunca".
func lastSold(at *time.Time) sheet.Cell {
	if at == nil {
		return sheet.Text("Nunca")
	}
	return sheet.Text(at.In(bogota).Format(time.DateOnly))
}
