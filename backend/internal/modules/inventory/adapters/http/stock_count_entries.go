package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http/inventoryapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// countManagePermission lets someone see and delete what everybody counted.
const countManagePermission = "inventory:count_manage"

// CountEntryUseCases are the use cases of what counters count.
type CountEntryUseCases interface {
	RecordCountEntry(ctx context.Context, a app.Actor, in app.CountEntryInput) (app.EntryView, error)
	DeleteCountEntry(ctx context.Context, a app.Actor, countID, entryID uuid.UUID, manages bool) error
	ListCountEntries(ctx context.Context, a app.Actor, f domain.EntryFilter, manages bool) (app.EntryPage, error)
}

// entryHandler serves what counters count.
type entryHandler struct {
	uc  CountEntryUseCases
	can auth.Authorizer
}

// manages reports whether the caller manages stock counts.
func (h entryHandler) manages(ctx context.Context) bool {
	p, ok := auth.PrincipalFrom(ctx)
	return ok && h.can(p.Role, countManagePermission)
}

// RecordCountEntry stores what the caller counted.
func (h entryHandler) RecordCountEntry(ctx context.Context, req inventoryapi.RecordCountEntryRequestObject) (inventoryapi.RecordCountEntryResponseObject, error) {
	qty, err := httpx.ParseDecimal(req.Body.Quantity, domain.ErrInvalidQuantity)
	if err != nil {
		return nil, err
	}
	e, err := h.uc.RecordCountEntry(ctx, actor(ctx), app.CountEntryInput{
		CountID: req.StockCountId, VariantID: req.Body.VariantId, Quantity: qty, UnitID: req.Body.UnitId,
	})
	if err != nil {
		return nil, err
	}
	return inventoryapi.RecordCountEntry201JSONResponse(entryBody(e)), nil
}

// DeleteCountEntry removes something counted.
func (h entryHandler) DeleteCountEntry(ctx context.Context, req inventoryapi.DeleteCountEntryRequestObject) (inventoryapi.DeleteCountEntryResponseObject, error) {
	if err := h.uc.DeleteCountEntry(ctx, actor(ctx), req.StockCountId, req.CountEntryId, h.manages(ctx)); err != nil {
		return nil, err
	}
	return inventoryapi.DeleteCountEntry204Response{}, nil
}

// ListCountEntries returns one page of what was counted.
func (h entryHandler) ListCountEntries(ctx context.Context, req inventoryapi.ListCountEntriesRequestObject) (inventoryapi.ListCountEntriesResponseObject, error) {
	f := domain.EntryFilter{CountID: req.StockCountId, VariantID: req.Params.VariantId}
	f.Limit, f.Offset = page(req.Params.Limit, req.Params.Offset)
	result, err := h.uc.ListCountEntries(ctx, actor(ctx), f, h.manages(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]inventoryapi.CountEntry, 0, len(result.Items))
	for _, e := range result.Items {
		items = append(items, entryBody(e))
	}
	return inventoryapi.ListCountEntries200JSONResponse{Items: items, Total: result.Total}, nil
}

func entryBody(e app.EntryView) inventoryapi.CountEntry {
	return inventoryapi.CountEntry{
		Id: e.ID(), VariantId: e.VariantID(), ProductId: e.ProductID,
		Quantity: httpx.FormatDecimal(e.Quantity()), CountedBy: e.CountedBy(), CountedAt: e.CountedAt(),
	}
}
