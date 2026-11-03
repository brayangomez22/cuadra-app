package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/http/salesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// ListCashSessions lists the tenant's cash sessions, with their cashier.
func (h *Handler) ListCashSessions(ctx context.Context, req salesapi.ListCashSessionsRequestObject) (salesapi.ListCashSessionsResponseObject, error) {
	p := req.Params
	q := app.CashSessionQuery{LocationID: p.LocationId, Limit: defaultPageSize}
	if p.Status != nil && *p.Status != salesapi.ListCashSessionsParamsStatusAll {
		q.Status = string(*p.Status)
	}
	if p.From != nil {
		q.From = &p.From.Time
	}
	if p.To != nil {
		q.To = &p.To.Time
	}
	if p.Limit != nil {
		q.Limit = *p.Limit
	}
	if p.Offset != nil {
		q.Offset = *p.Offset
	}
	page, err := h.uc.ListCashSessions(ctx, h.actor(ctx), q)
	if err != nil {
		return nil, err
	}
	body := salesapi.ListCashSessions200JSONResponse{
		Items: make([]salesapi.CashSessionSummary, 0, len(page.Items)), Total: page.Total,
	}
	for _, row := range page.Items {
		body.Items = append(body.Items, summaryBody(row))
	}
	return body, nil
}

func summaryBody(row app.CashSessionRow) salesapi.CashSessionSummary {
	body := salesapi.CashSessionSummary{
		Id: row.ID, LocationId: row.LocationID, UserId: row.UserID,
		Status: salesapi.CashSessionSummaryStatus(row.Status), OpeningFloat: httpx.FormatMoney(row.OpeningFloat),
		OpenedAt: row.OpenedAt, ClosedAt: row.ClosedAt, ClosedBy: row.ClosedBy,
	}
	if row.CashierName != "" {
		body.CashierName = &row.CashierName
	}
	if result, ok := row.Result(); ok {
		body.Result = new(salesapi.CashSessionSummaryResult(result))
		body.Difference = new(httpx.FormatMoney(*row.Difference))
		body.SalesCount = &row.SalesCount
		body.SalesTotal = new(httpx.FormatMoney(row.SalesTotal))
	}
	return body
}
