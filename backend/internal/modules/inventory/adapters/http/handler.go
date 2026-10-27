// Package httpadapter exposes the inventory use cases over HTTP: it implements
// the strict server generated from the "inventory" operations of
// api/openapi.yaml (inventoryapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http/inventoryapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// UseCases are the inventory use cases the handlers call (*app.Service).
type UseCases interface {
	CreateLocation(ctx context.Context, a app.Actor, name string) (*domain.Location, error)
	ListLocations(ctx context.Context, a app.Actor) ([]*domain.Location, error)
	ReceiveStock(ctx context.Context, a app.Actor, in app.ReceiptInput) (app.Movement, error)
	AdjustStock(ctx context.Context, a app.Actor, in app.AdjustmentInput) (app.Movement, error)
	TransferStock(ctx context.Context, a app.Actor, in app.TransferInput) (out, inMov app.Movement, err error)
	ListStock(ctx context.Context, a app.Actor, q app.StockQuery) (app.LevelPage, error)
	GetKardex(ctx context.Context, a app.Actor, q app.KardexQuery) (app.MovementPage, error)
	SetReorderRule(ctx context.Context, a app.Actor, in app.ReorderInput) (*domain.ReorderRule, error)
	DeleteReorderRule(ctx context.Context, a app.Actor, variantID, locationID uuid.UUID) error
	ListReorderAlerts(ctx context.Context, a app.Actor, q app.ReorderQuery) (app.AlertPage, error)
}

// Handler implements inventoryapi.StrictServerInterface.
type Handler struct {
	uc  UseCases
	can auth.Authorizer
}

var _ inventoryapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the inventory operations on mux, with the module's error
// mapping. can decides who sees costs.
func Register(mux *http.ServeMux, uc UseCases, can auth.Authorizer, log *slog.Logger) {
	h := &Handler{uc: uc, can: can}
	errs := httpx.NewErrorMapper(log, errorRules...)
	inventoryapi.HandlerWithOptions(
		inventoryapi.NewStrictHandlerWithOptions(h, []inventoryapi.StrictMiddlewareFunc{auth.RequirePrincipal[inventoryapi.StrictHandlerFunc]}, inventoryapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		inventoryapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor returns who calls: tenant and user come only from the access token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}
}

// seesCost reports whether the caller may see costs and margins.
func (h *Handler) seesCost(ctx context.Context) bool { return auth.SeesCosts(ctx, h.can) }

// page returns the contract's limit and offset, with their defaults.
func page(limit, offset *int) (int, int) {
	l, o := defaultPageSize, 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}
	return l, o
}

// ListLocations returns every location of the tenant.
func (h *Handler) ListLocations(ctx context.Context, _ inventoryapi.ListLocationsRequestObject) (inventoryapi.ListLocationsResponseObject, error) {
	list, err := h.uc.ListLocations(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]inventoryapi.Location, 0, len(list))
	for _, l := range list {
		items = append(items, locationBody(l))
	}
	return inventoryapi.ListLocations200JSONResponse{Items: items}, nil
}

// CreateLocation adds a location.
func (h *Handler) CreateLocation(ctx context.Context, req inventoryapi.CreateLocationRequestObject) (inventoryapi.CreateLocationResponseObject, error) {
	l, err := h.uc.CreateLocation(ctx, actor(ctx), req.Body.Name)
	if err != nil {
		return nil, err
	}
	return inventoryapi.CreateLocation201JSONResponse(locationBody(l)), nil
}

// ListStock returns the stock at a location.
func (h *Handler) ListStock(ctx context.Context, req inventoryapi.ListStockRequestObject) (inventoryapi.ListStockResponseObject, error) {
	q := app.StockQuery{LocationID: req.LocationId, VariantID: req.Params.VariantId}
	q.Limit, q.Offset = page(req.Params.Limit, req.Params.Offset)
	result, err := h.uc.ListStock(ctx, actor(ctx), q)
	if errors.Is(err, domain.ErrLocationNotFound) {
		return inventoryapi.ListStock404JSONResponse{NotFoundJSONResponse: inventoryapi.NotFoundJSONResponse{
			Error: inventoryapi.ErrorDetail{Code: locationNotFoundCode, Message: locationNotFoundMessage},
		}}, nil
	}
	if err != nil {
		return nil, err
	}
	seesCost := h.seesCost(ctx)
	items := make([]inventoryapi.StockLevel, 0, len(result.Items))
	for _, l := range result.Items {
		items = append(items, levelBody(l, seesCost))
	}
	return inventoryapi.ListStock200JSONResponse{Items: items, Total: result.Total}, nil
}

// GetVariantKardex returns a variant's movements, newest first.
func (h *Handler) GetVariantKardex(ctx context.Context, req inventoryapi.GetVariantKardexRequestObject) (inventoryapi.GetVariantKardexResponseObject, error) {
	q := app.KardexQuery{VariantID: req.VariantId, LocationID: req.Params.LocationId}
	q.Limit, q.Offset = page(req.Params.Limit, req.Params.Offset)
	body, err := h.kardex(ctx, q)
	if err != nil {
		return nil, err
	}
	return inventoryapi.GetVariantKardex200JSONResponse(body), nil
}

func (h *Handler) kardex(ctx context.Context, q app.KardexQuery) (inventoryapi.StockMovementPage, error) {
	result, err := h.uc.GetKardex(ctx, actor(ctx), q)
	if err != nil {
		return inventoryapi.StockMovementPage{}, err
	}
	return inventoryapi.StockMovementPage{Items: h.movementBodies(ctx, result.Items...), Total: result.Total}, nil
}

// movementBodies writes movements, with their costs only for whoever sees
// costs.
func (h *Handler) movementBodies(ctx context.Context, movements ...app.Movement) []inventoryapi.StockMovement {
	seesCost := h.seesCost(ctx)
	bodies := make([]inventoryapi.StockMovement, 0, len(movements))
	for _, m := range movements {
		bodies = append(bodies, movementBody(m, seesCost))
	}
	return bodies
}

// ReceiveStock registers an entry of goods.
func (h *Handler) ReceiveStock(ctx context.Context, req inventoryapi.ReceiveStockRequestObject) (inventoryapi.ReceiveStockResponseObject, error) {
	qty, err := httpx.ParseDecimal(req.Body.Quantity, domain.ErrInvalidQuantity)
	if err != nil {
		return nil, err
	}
	cost, err := httpx.ParseDecimal(req.Body.UnitCost, domain.ErrInvalidUnitCost)
	if err != nil {
		return nil, err
	}
	m, err := h.uc.ReceiveStock(ctx, actor(ctx), app.ReceiptInput{
		VariantID: req.Body.VariantId, LocationID: req.Body.LocationId, Quantity: qty, UnitCost: cost,
		UnitID: req.Body.UnitId,
	})
	if err != nil {
		return nil, err
	}
	return inventoryapi.ReceiveStock201JSONResponse(h.movementBodies(ctx, m)[0]), nil
}

// AdjustStock registers a correction of the stock.
func (h *Handler) AdjustStock(ctx context.Context, req inventoryapi.AdjustStockRequestObject) (inventoryapi.AdjustStockResponseObject, error) {
	qty, err := httpx.ParseDecimal(req.Body.Quantity, domain.ErrInvalidQuantity)
	if err != nil {
		return nil, err
	}
	m, err := h.uc.AdjustStock(ctx, actor(ctx), app.AdjustmentInput{
		VariantID: req.Body.VariantId, LocationID: req.Body.LocationId, Quantity: qty, Reason: req.Body.Reason,
	})
	if err != nil {
		return nil, err
	}
	return inventoryapi.AdjustStock201JSONResponse(h.movementBodies(ctx, m)[0]), nil
}

// TransferStock moves stock between two locations.
func (h *Handler) TransferStock(ctx context.Context, req inventoryapi.TransferStockRequestObject) (inventoryapi.TransferStockResponseObject, error) {
	qty, err := httpx.ParseDecimal(req.Body.Quantity, domain.ErrInvalidQuantity)
	if err != nil {
		return nil, err
	}
	out, in, err := h.uc.TransferStock(ctx, actor(ctx), app.TransferInput{
		VariantID: req.Body.VariantId, FromLocationID: req.Body.FromLocationId, ToLocationID: req.Body.ToLocationId, Quantity: qty,
	})
	if err != nil {
		return nil, err
	}
	bodies := h.movementBodies(ctx, out, in)
	return inventoryapi.TransferStock201JSONResponse{Out: bodies[0], In: bodies[1]}, nil
}

func locationBody(l *domain.Location) inventoryapi.Location {
	return inventoryapi.Location{Id: l.ID(), Name: l.Name(), Active: l.IsActive(), CreatedAt: l.CreatedAt()}
}

// levelBody writes a level; the average cost only for whoever sees costs.
func levelBody(l app.Level, seesCost bool) inventoryapi.StockLevel {
	body := inventoryapi.StockLevel{
		VariantId: l.VariantID(), ProductId: l.ProductID,
		LocationId: l.LocationID(), Quantity: httpx.FormatDecimal(l.Quantity()),
		Reorder: stockReorder(l),
	}
	if seesCost {
		cost := httpx.FormatMoney(l.AverageCost())
		body.AverageCost = &cost
	}
	return body
}

// movementBody writes a movement; its costs only for whoever sees costs.
func movementBody(m app.Movement, seesCost bool) inventoryapi.StockMovement {
	body := inventoryapi.StockMovement{
		Id:           m.ID(),
		VariantId:    m.VariantID(),
		ProductId:    m.ProductID,
		LocationId:   m.LocationID(),
		Type:         inventoryapi.StockMovementType(m.Type()),
		Quantity:     httpx.FormatDecimal(m.Quantity()),
		BalanceAfter: httpx.FormatDecimal(m.BalanceAfter()),
		UserId:       m.UserID(),
		OccurredAt:   m.OccurredAt(),
	}
	if seesCost {
		unitCost, avg := httpx.FormatMoney(m.UnitCost()), httpx.FormatMoney(m.AverageCostAfter())
		body.UnitCost, body.AverageCostAfter = &unitCost, &avg
	}
	if reason := m.Reason(); reason != "" {
		body.Reason = &reason
	}
	if ref := m.Reference(); ref != nil {
		body.Reference = &inventoryapi.DocumentRef{Kind: inventoryapi.DocumentRefKind(ref.Kind), Id: ref.ID}
	}
	return body
}
