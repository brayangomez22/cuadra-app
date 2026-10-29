package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/http/receivablesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
)

// ReportUseCases are the reports of the tenant's receivables.
type ReportUseCases interface {
	Aging(ctx context.Context, a app.Actor, limit, offset int) (app.AgingReport, error)
}

// reportHandlers are the reports of the receivables.
type reportHandlers struct{ uc ReportUseCases }

// GetReceivablesAging returns a page of the aging by customer.
func (h *reportHandlers) GetReceivablesAging(ctx context.Context, req receivablesapi.GetReceivablesAgingRequestObject) (receivablesapi.GetReceivablesAgingResponseObject, error) {
	limit, offset := page(req.Params.Limit, req.Params.Offset)
	report, err := h.uc.Aging(ctx, actor(ctx), limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]receivablesapi.AgingRow, 0, len(report.Rows))
	for _, r := range report.Rows {
		items = append(items, receivablesapi.AgingRow{CustomerId: r.CustomerID, Name: r.Name, Buckets: bucketsBody(r.Aging)})
	}
	return receivablesapi.GetReceivablesAging200JSONResponse{
		AsOf: date(report.AsOf), Items: items, Total: report.Total, Totals: bucketsBody(report.Totals),
	}, nil
}
