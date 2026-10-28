package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http/inventoryapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// CountUseCases are the use cases of the stock counts themselves.
type CountUseCases interface {
	StartStockCount(ctx context.Context, a app.Actor, in app.StartCountInput) (app.CountView, error)
	GetStockCount(ctx context.Context, a app.Actor, id uuid.UUID) (app.CountView, error)
	ListStockCounts(ctx context.Context, a app.Actor, f domain.CountFilter) (domain.StockCountPage, error)
	StockCountLines(ctx context.Context, a app.Actor, id uuid.UUID, f app.LineFilter) (app.CountLinesPage, error)
	RequestRecount(ctx context.Context, a app.Actor, id uuid.UUID, variantIDs []uuid.UUID) error
	ApproveStockCount(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.StockCount, error)
	CancelStockCount(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.StockCount, error)
}

// countHandler serves the stock counts.
type countHandler struct {
	uc  CountUseCases
	can auth.Authorizer
}

// StartStockCount opens a count; without a mode, counts of the same variant
// add up.
func (h countHandler) StartStockCount(ctx context.Context, req inventoryapi.StartStockCountRequestObject) (inventoryapi.StartStockCountResponseObject, error) {
	in := app.StartCountInput{LocationID: req.Body.LocationId, Mode: domain.CountSum}
	if req.Body.CategoryIds != nil {
		in.CategoryIDs = *req.Body.CategoryIds
	}
	if req.Body.Mode != nil {
		in.Mode = domain.CountMode(*req.Body.Mode)
	}
	c, err := h.uc.StartStockCount(ctx, actor(ctx), in)
	if err != nil {
		return nil, err
	}
	return inventoryapi.StartStockCount201JSONResponse(countBody(c.StockCount, c.Progress, auth.SeesCosts(ctx, h.can))), nil
}

// GetStockCount returns a count with its progress.
func (h countHandler) GetStockCount(ctx context.Context, req inventoryapi.GetStockCountRequestObject) (inventoryapi.GetStockCountResponseObject, error) {
	c, err := h.uc.GetStockCount(ctx, actor(ctx), req.StockCountId)
	if err != nil {
		return nil, err
	}
	return inventoryapi.GetStockCount200JSONResponse(countBody(c.StockCount, c.Progress, auth.SeesCosts(ctx, h.can))), nil
}

// ListStockCounts returns one page of counts. A list carries no progress:
// the approved counts carry their summary, which has the same numbers.
func (h countHandler) ListStockCounts(ctx context.Context, req inventoryapi.ListStockCountsRequestObject) (inventoryapi.ListStockCountsResponseObject, error) {
	f := domain.CountFilter{LocationID: req.Params.LocationId}
	if req.Params.Status != nil {
		status := domain.CountStatus(*req.Params.Status)
		f.Status = &status
	}
	f.Limit, f.Offset = page(req.Params.Limit, req.Params.Offset)
	result, err := h.uc.ListStockCounts(ctx, actor(ctx), f)
	if err != nil {
		return nil, err
	}
	seesCost := auth.SeesCosts(ctx, h.can)
	items := make([]inventoryapi.StockCount, 0, len(result.Items))
	for _, c := range result.Items {
		items = append(items, countBody(c, listedProgress(c), seesCost))
	}
	return inventoryapi.ListStockCounts200JSONResponse{Items: items, Total: result.Total}, nil
}

// listedProgress is the progress a listed count shows: its summary's when
// it was approved, none otherwise.
func listedProgress(c *domain.StockCount) domain.CountProgress {
	summary, _ := c.Summary()
	return domain.CountProgress{Lines: summary.Lines, Counted: summary.Counted}
}

// ApproveStockCount closes a count and adjusts the stock.
func (h countHandler) ApproveStockCount(ctx context.Context, req inventoryapi.ApproveStockCountRequestObject) (inventoryapi.ApproveStockCountResponseObject, error) {
	c, err := h.uc.ApproveStockCount(ctx, actor(ctx), req.StockCountId)
	if err != nil {
		return nil, err
	}
	return inventoryapi.ApproveStockCount200JSONResponse(countBody(c, listedProgress(c), auth.SeesCosts(ctx, h.can))), nil
}

// CancelStockCount closes a count without adjusting the stock.
func (h countHandler) CancelStockCount(ctx context.Context, req inventoryapi.CancelStockCountRequestObject) (inventoryapi.CancelStockCountResponseObject, error) {
	c, err := h.uc.CancelStockCount(ctx, actor(ctx), req.StockCountId)
	if err != nil {
		return nil, err
	}
	return inventoryapi.CancelStockCount200JSONResponse(countBody(c, listedProgress(c), false)), nil
}
