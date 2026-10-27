package httpadapter

import (
	"context"
	"errors"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http/inventoryapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// SetReorderRule stores a variant's minimum and maximum at a location. The
// location and the variant are the path's resources: missing, they are 404.
func (h *Handler) SetReorderRule(ctx context.Context, req inventoryapi.SetReorderRuleRequestObject) (inventoryapi.SetReorderRuleResponseObject, error) {
	minQty, err := httpx.ParseDecimal(req.Body.MinQuantity, domain.ErrInvalidReorderRange)
	if err != nil {
		return nil, err
	}
	maxQty, err := httpx.ParseDecimal(req.Body.MaxQuantity, domain.ErrInvalidReorderRange)
	if err != nil {
		return nil, err
	}
	rule, err := h.uc.SetReorderRule(ctx, actor(ctx), app.ReorderInput{
		VariantID: req.VariantId, LocationID: req.LocationId, Min: minQty, Max: maxQty,
	})
	switch {
	case errors.Is(err, domain.ErrLocationNotFound):
		return inventoryapi.SetReorderRule404JSONResponse{Error: inventoryapi.ErrorDetail{Code: locationNotFoundCode, Message: locationNotFoundMessage}}, nil
	case errors.Is(err, domain.ErrVariantNotFound):
		return inventoryapi.SetReorderRule404JSONResponse{Error: inventoryapi.ErrorDetail{Code: variantNotFoundCode, Message: variantNotFoundMessage}}, nil
	case err != nil:
		return nil, err
	}
	return inventoryapi.SetReorderRule200JSONResponse{
		VariantId: rule.VariantID(), LocationId: rule.LocationID(),
		MinQuantity: httpx.FormatDecimal(rule.Min()), MaxQuantity: httpx.FormatDecimal(rule.Max()),
	}, nil
}

// DeleteReorderRule removes a variant's minimum and maximum at a location.
func (h *Handler) DeleteReorderRule(ctx context.Context, req inventoryapi.DeleteReorderRuleRequestObject) (inventoryapi.DeleteReorderRuleResponseObject, error) {
	if err := h.uc.DeleteReorderRule(ctx, actor(ctx), req.VariantId, req.LocationId); err != nil {
		return nil, err
	}
	return inventoryapi.DeleteReorderRule204Response{}, nil
}

// ListReorderAlerts returns the variants at or below their minimum.
func (h *Handler) ListReorderAlerts(ctx context.Context, req inventoryapi.ListReorderAlertsRequestObject) (inventoryapi.ListReorderAlertsResponseObject, error) {
	q := app.ReorderQuery{LocationID: req.Params.LocationId}
	q.Limit, q.Offset = page(req.Params.Limit, req.Params.Offset)
	result, err := h.uc.ListReorderAlerts(ctx, actor(ctx), q)
	if err != nil {
		return nil, err
	}
	items := make([]inventoryapi.ReorderAlert, 0, len(result.Items))
	for _, a := range result.Items {
		items = append(items, inventoryapi.ReorderAlert{
			VariantId: a.Rule.VariantID(), ProductId: a.ProductID, LocationId: a.Rule.LocationID(),
			Quantity:          httpx.FormatDecimal(a.OnHand),
			MinQuantity:       httpx.FormatDecimal(a.Rule.Min()),
			MaxQuantity:       httpx.FormatDecimal(a.Rule.Max()),
			SuggestedQuantity: httpx.FormatDecimal(a.Suggested),
		})
	}
	return inventoryapi.ListReorderAlerts200JSONResponse{Items: items, Total: result.Total}, nil
}

// stockReorder writes a level's reorder rule; nil without one.
func stockReorder(l app.Level) *inventoryapi.StockReorder {
	if l.Reorder == nil {
		return nil
	}
	return &inventoryapi.StockReorder{
		MinQuantity:  httpx.FormatDecimal(l.Reorder.Min()),
		MaxQuantity:  httpx.FormatDecimal(l.Reorder.Max()),
		NeedsReorder: l.NeedsReorder(),
	}
}
