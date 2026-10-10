// Package httpadapter exposes the inventory use cases over HTTP: it implements
// the strict server generated from the "inventory" operations of
// api/openapi.yaml (inventoryapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http/inventoryapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// permissionCatalogWrite lets a role see costs. It is the identity module's
// "catalog:write", named here as the contract names it.
const permissionCatalogWrite = "catalog:write"

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// UseCases are the inventory use cases the handlers call (*app.Service).
type UseCases interface {
	CreateLocation(ctx context.Context, a app.Actor, name string) (*domain.Location, error)
	ListLocations(ctx context.Context, a app.Actor) ([]*domain.Location, error)
	ReceiveStock(ctx context.Context, a app.Actor, in app.ReceiptInput) (*domain.StockMovement, error)
	AdjustStock(ctx context.Context, a app.Actor, in app.AdjustmentInput) (*domain.StockMovement, error)
	TransferStock(ctx context.Context, a app.Actor, in app.TransferInput) (out, inMov *domain.StockMovement, err error)
	ListStock(ctx context.Context, a app.Actor, f domain.StockFilter) (domain.StockLevelPage, error)
	GetKardex(ctx context.Context, a app.Actor, f domain.KardexFilter) (domain.MovementPage, error)
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
		inventoryapi.NewStrictHandlerWithOptions(h, []inventoryapi.StrictMiddlewareFunc{requirePrincipal}, inventoryapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		inventoryapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// requirePrincipal answers 401 when no principal reached the handlers. The
// API's auth middleware already demands one; this keeps a wiring mistake
// from running a use case without a tenant.
func requirePrincipal(next inventoryapi.StrictHandlerFunc, _ string) inventoryapi.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
		if _, ok := auth.PrincipalFrom(ctx); !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", unauthorizedMessage)
			return nil, nil
		}
		return next(ctx, w, r, req)
	}
}

// actor returns who calls: tenant and user come only from the access token.
func actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID}
}

// seesCost reports whether the caller may see costs.
func (h *Handler) seesCost(ctx context.Context) bool {
	p, _ := auth.PrincipalFrom(ctx)
	return h.can(p.Role, permissionCatalogWrite)
}

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
	f := domain.StockFilter{LocationID: req.LocationId, ProductID: req.Params.ProductId}
	f.Limit, f.Offset = page(req.Params.Limit, req.Params.Offset)
	result, err := h.uc.ListStock(ctx, actor(ctx), f)
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

// GetKardex returns a product's movements, newest first.
func (h *Handler) GetKardex(ctx context.Context, req inventoryapi.GetKardexRequestObject) (inventoryapi.GetKardexResponseObject, error) {
	f := domain.KardexFilter{ProductID: req.ProductId, LocationID: req.Params.LocationId}
	f.Limit, f.Offset = page(req.Params.Limit, req.Params.Offset)
	result, err := h.uc.GetKardex(ctx, actor(ctx), f)
	if err != nil {
		return nil, err
	}
	seesCost := h.seesCost(ctx)
	items := make([]inventoryapi.StockMovement, 0, len(result.Items))
	for _, m := range result.Items {
		items = append(items, movementBody(m, seesCost))
	}
	return inventoryapi.GetKardex200JSONResponse{Items: items, Total: result.Total}, nil
}

// ReceiveStock registers an entry of goods.
func (h *Handler) ReceiveStock(ctx context.Context, req inventoryapi.ReceiveStockRequestObject) (inventoryapi.ReceiveStockResponseObject, error) {
	qty, err := parseDecimal(req.Body.Quantity, domain.ErrInvalidQuantity)
	if err != nil {
		return nil, err
	}
	cost, err := parseDecimal(req.Body.UnitCost, domain.ErrInvalidUnitCost)
	if err != nil {
		return nil, err
	}
	m, err := h.uc.ReceiveStock(ctx, actor(ctx), app.ReceiptInput{
		ProductID: req.Body.ProductId, LocationID: req.Body.LocationId, Quantity: qty, UnitCost: cost,
	})
	if err != nil {
		return nil, err
	}
	return inventoryapi.ReceiveStock201JSONResponse(movementBody(m, h.seesCost(ctx))), nil
}

// AdjustStock registers a correction of the stock.
func (h *Handler) AdjustStock(ctx context.Context, req inventoryapi.AdjustStockRequestObject) (inventoryapi.AdjustStockResponseObject, error) {
	qty, err := parseDecimal(req.Body.Quantity, domain.ErrInvalidQuantity)
	if err != nil {
		return nil, err
	}
	m, err := h.uc.AdjustStock(ctx, actor(ctx), app.AdjustmentInput{
		ProductID: req.Body.ProductId, LocationID: req.Body.LocationId, Quantity: qty, Reason: req.Body.Reason,
	})
	if err != nil {
		return nil, err
	}
	return inventoryapi.AdjustStock201JSONResponse(movementBody(m, h.seesCost(ctx))), nil
}

// TransferStock moves stock between two locations.
func (h *Handler) TransferStock(ctx context.Context, req inventoryapi.TransferStockRequestObject) (inventoryapi.TransferStockResponseObject, error) {
	qty, err := parseDecimal(req.Body.Quantity, domain.ErrInvalidQuantity)
	if err != nil {
		return nil, err
	}
	out, in, err := h.uc.TransferStock(ctx, actor(ctx), app.TransferInput{
		ProductID: req.Body.ProductId, FromLocationID: req.Body.FromLocationId, ToLocationID: req.Body.ToLocationId, Quantity: qty,
	})
	if err != nil {
		return nil, err
	}
	seesCost := h.seesCost(ctx)
	return inventoryapi.TransferStock201JSONResponse{Out: movementBody(out, seesCost), In: movementBody(in, seesCost)}, nil
}

func locationBody(l *domain.Location) inventoryapi.Location {
	return inventoryapi.Location{Id: l.ID(), Name: l.Name(), Active: l.IsActive(), CreatedAt: l.CreatedAt()}
}

// levelBody writes a level; the average cost only for whoever sees costs.
func levelBody(l *domain.StockLevel, seesCost bool) inventoryapi.StockLevel {
	body := inventoryapi.StockLevel{ProductId: l.ProductID(), LocationId: l.LocationID(), Quantity: formatQuantity(l.Quantity())}
	if seesCost {
		cost := formatMoney(l.AverageCost())
		body.AverageCost = &cost
	}
	return body
}

// movementBody writes a movement; its costs only for whoever sees costs.
func movementBody(m *domain.StockMovement, seesCost bool) inventoryapi.StockMovement {
	body := inventoryapi.StockMovement{
		Id:           m.ID(),
		ProductId:    m.ProductID(),
		LocationId:   m.LocationID(),
		Type:         inventoryapi.StockMovementType(m.Type()),
		Quantity:     formatQuantity(m.Quantity()),
		BalanceAfter: formatQuantity(m.BalanceAfter()),
		UserId:       m.UserID(),
		OccurredAt:   m.OccurredAt(),
	}
	if seesCost {
		unitCost, avg := formatMoney(m.UnitCost()), formatMoney(m.AverageCostAfter())
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
