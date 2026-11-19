package httpadapter

import (
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http/reportsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// money is an amount in pesos, rounded to cents.
func money(d decimal.Decimal) string { return httpx.FormatMoney(domain.Money(d)) }

// optionalMoney is money, or nil when costs are hidden.
func optionalMoney(d decimal.Decimal, costs bool) *string {
	if !costs {
		return nil
	}
	m := money(d)
	return &m
}

// quantity is a quantity in the base unit.
func quantity(d decimal.Decimal) string { return httpx.FormatDecimal(d.Round(4)) }

// date is a day of the contract.
func date(d calendar.Date) openapi_types.Date { return openapi_types.Date{Time: d.Time()} }

// day reads a day of the contract.
func day(d openapi_types.Date) calendar.Date { return calendar.DateOf(d.Time) }

// days is the period from..to of a request.
func days(from, to openapi_types.Date) app.Range { return app.Range{From: day(from), To: day(to)} }

func period(p domain.Period) reportsapi.ReportPeriod {
	return reportsapi.ReportPeriod{From: date(p.From()), To: date(p.To())}
}

// profit is the figures of what was sold: revenue, and cost, profit and
// margin for whoever sees costs (margin only with revenue).
func profit(p domain.Profit, costs bool) reportsapi.ProfitFigures {
	f := reportsapi.ProfitFigures{Revenue: money(p.Revenue)}
	if !costs {
		return f
	}
	f.Cost = optionalMoney(p.Cost, true)
	amount := money(p.Amount())
	f.Profit = &amount
	if m, ok := p.Margin(); ok {
		s := m.StringFixed(4)
		f.Margin = &s
	}
	return f
}

// rate is a rate with 4 decimals, or nil without one.
func rate(r decimal.Decimal, ok bool) *string {
	if !ok {
		return nil
	}
	s := r.StringFixed(4)
	return &s
}
