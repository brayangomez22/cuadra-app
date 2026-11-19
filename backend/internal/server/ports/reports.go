package ports

import (
	"context"

	"github.com/google/uuid"

	customersapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables"
	receivablesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	reportsdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

// ReportsReceivables satisfies reportsapp.Receivables with the receivables
// module's aging: the reports export it without repeating its rules.
type ReportsReceivables struct{ Receivables *receivables.Module }

// Aging returns every customer's aging as of today, in sort's order.
func (r ReportsReceivables) Aging(ctx context.Context, tenantID, userID uuid.UUID, sort string) (reportsdomain.AgingExport, error) {
	report, err := r.Receivables.AgingAll(ctx, tenantID, userID, sort)
	if err != nil {
		return reportsdomain.AgingExport{}, err
	}
	out := reportsdomain.AgingExport{AsOf: report.AsOf, Lines: make([]reportsdomain.AgingLine, 0, len(report.Rows))}
	for _, row := range report.Rows {
		out.Lines = append(out.Lines, agingLine(row.Name, row.Aging))
	}
	out.Totals = agingLine("", report.Totals)
	return out, nil
}

func agingLine(name string, g *receivablesdomain.Aging) reportsdomain.AgingLine {
	return reportsdomain.AgingLine{
		Name: name, Current: g.Current, Days1To30: g.Days1To30, Days31To60: g.Days31To60, Days61To90: g.Days61To90,
		Over90: g.Over90, Overdue: g.Overdue(), Total: g.Total(),
	}
}

// CustomerBalances satisfies customersapp.CustomerBalances with what the
// receivables module says each customer owes. The receivables module reads
// the customers' names, so Receivables is set once both exist.
type CustomerBalances struct{ Receivables *receivables.Module }

// Balances returns what each of the customers owes.
func (c *CustomerBalances) Balances(ctx context.Context, tenantID, userID uuid.UUID, customerIDs []uuid.UUID) (map[uuid.UUID]customersapp.Balance, error) {
	owed, err := c.Receivables.Balances(ctx, tenantID, userID, customerIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]customersapp.Balance, len(owed))
	for id, b := range owed {
		out[id] = customersapp.Balance{Balance: b.Balance, Overdue: b.Overdue}
	}
	return out, nil
}
