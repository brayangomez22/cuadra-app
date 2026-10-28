package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http/inventoryapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// ListStockCountLines returns one page of the evaluated lines of a count.
func (h countHandler) ListStockCountLines(ctx context.Context, req inventoryapi.ListStockCountLinesRequestObject) (inventoryapi.ListStockCountLinesResponseObject, error) {
	var f app.LineFilter
	if req.Params.Status != nil {
		status := domain.LineStatus(*req.Params.Status)
		f.Status = &status
	}
	f.Limit, f.Offset = page(req.Params.Limit, req.Params.Offset)
	result, err := h.uc.StockCountLines(ctx, actor(ctx), req.StockCountId, f)
	if err != nil {
		return nil, err
	}
	seesCost := auth.SeesCosts(ctx, h.can)
	items := make([]inventoryapi.StockCountLine, 0, len(result.Items))
	for _, l := range result.Items {
		items = append(items, lineBody(l, seesCost))
	}
	return inventoryapi.ListStockCountLines200JSONResponse{
		Items: items, Total: result.Total, Summary: summaryBody(result.Summary, seesCost),
	}, nil
}

// RequestRecount starts a new round of the given variants.
func (h countHandler) RequestRecount(ctx context.Context, req inventoryapi.RequestRecountRequestObject) (inventoryapi.RequestRecountResponseObject, error) {
	if err := h.uc.RequestRecount(ctx, actor(ctx), req.StockCountId, req.Body.VariantIds); err != nil {
		return nil, err
	}
	return inventoryapi.RequestRecount204Response{}, nil
}

// lineBody is an evaluated line. What was counted is left out while nobody
// counted it or it is under review, and its value without cost permission.
func lineBody(l app.LineView, seesCost bool) inventoryapi.StockCountLine {
	body := inventoryapi.StockCountLine{
		VariantId: l.VariantID, ProductId: l.ProductID,
		Status:           inventoryapi.StockCountLineStatus(l.Status),
		ExpectedQuantity: httpx.FormatDecimal(l.Expected),
	}
	if l.Status == domain.LineUncounted || l.Status == domain.LineReview {
		return body
	}
	counted, difference := httpx.FormatDecimal(l.Counted), httpx.FormatDecimal(l.Difference)
	body.CountedQuantity, body.Difference = &counted, &difference
	if seesCost {
		value := httpx.FormatMoney(l.Value)
		body.DifferenceValue = &value
	}
	return body
}

// countBody is a count with its progress and, if approved, its summary.
func countBody(c *domain.StockCount, p domain.CountProgress, seesCost bool) inventoryapi.StockCount {
	body := inventoryapi.StockCount{
		Id: c.ID(), LocationId: c.LocationID(), CategoryIds: c.CategoryIDs(),
		Mode:      inventoryapi.StockCountMode(c.Mode()),
		Status:    inventoryapi.StockCountStatus(c.Status()),
		StartedBy: c.StartedBy(), StartedAt: c.StartedAt(),
		Lines: p.Lines, CountedLines: p.Counted,
	}
	if c.Status() != domain.CountOpen {
		closedBy, closedAt := c.ClosedBy(), c.ClosedAt()
		body.ClosedBy, body.ClosedAt = &closedBy, &closedAt
	}
	if summary, ok := c.Summary(); ok {
		s := summaryBody(summary, seesCost)
		body.Summary = &s
	}
	return body
}

// summaryBody is the accuracy of a count; its values only with cost
// permission.
func summaryBody(s domain.CountSummary, seesCost bool) inventoryapi.StockCountSummary {
	body := inventoryapi.StockCountSummary{Lines: s.Lines, CountedLines: s.Counted, ExactLines: s.Exact}
	if accuracy, ok := s.Accuracy(); ok {
		a := accuracy.StringFixed(4)
		body.Accuracy = &a
	}
	if seesCost {
		net, abs := httpx.FormatMoney(s.NetValue), httpx.FormatMoney(s.AbsoluteValue)
		body.NetDifferenceValue, body.AbsoluteDifferenceValue = &net, &abs
	}
	return body
}
