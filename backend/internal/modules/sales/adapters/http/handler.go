// Package httpadapter exposes the sales use cases over HTTP: it implements
// the strict server generated from the "sales" operations of
// api/openapi.yaml (salesapi) and only translates HTTP ↔ use case.
package httpadapter

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/http/salesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// permissionSalesDiscount lets a role discount without the tenant's limit.
// It is the identity module's "sales:discount", named here as the contract
// names it.
const permissionSalesDiscount = "sales:discount"

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// UseCases are the sales use cases the handlers call (*app.Service).
type UseCases interface {
	ConfirmSale(ctx context.Context, a app.Actor, in app.ConfirmSaleInput) (*domain.Sale, error)
	VoidSale(ctx context.Context, a app.Actor, id uuid.UUID, reason string) (*domain.Sale, error)
	GetSale(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Sale, error)
	ListSales(ctx context.Context, a app.Actor, in app.ListSalesInput) (domain.SalePage, error)
}

// Handler implements salesapi.StrictServerInterface.
type Handler struct {
	uc  UseCases
	can auth.Authorizer
}

var _ salesapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the sales operations on mux, with the module's error
// mapping. can decides who discounts without limit.
func Register(mux *http.ServeMux, uc UseCases, can auth.Authorizer, log *slog.Logger) {
	h := &Handler{uc: uc, can: can}
	errs := httpx.NewErrorMapper(log, errorRules...)
	salesapi.HandlerWithOptions(
		salesapi.NewStrictHandlerWithOptions(h, []salesapi.StrictMiddlewareFunc{requirePrincipal}, salesapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		salesapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// requirePrincipal answers 401 when no principal reached the handlers. The
// API's auth middleware already demands one; this keeps a wiring mistake
// from running a use case without a tenant.
func requirePrincipal(next salesapi.StrictHandlerFunc, _ string) salesapi.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
		if _, ok := auth.PrincipalFrom(ctx); !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", unauthorizedMessage)
			return nil, nil
		}
		return next(ctx, w, r, req)
	}
}

// actor returns who calls: tenant and user come only from the access token,
// and whether they discount without limit from their role.
func (h *Handler) actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID, UnlimitedDiscount: h.can(p.Role, permissionSalesDiscount)}
}

// ConfirmSale checks out a cart.
func (h *Handler) ConfirmSale(ctx context.Context, req salesapi.ConfirmSaleRequestObject) (salesapi.ConfirmSaleResponseObject, error) {
	in := app.ConfirmSaleInput{LocationID: req.Body.LocationId}
	for _, l := range req.Body.Lines {
		qty, err := parseDecimal(l.Quantity, domain.ErrInvalidQuantity)
		if err != nil {
			return nil, err
		}
		discount := decimal.Zero
		if l.Discount != nil {
			if discount, err = parseDecimal(*l.Discount, domain.ErrInvalidDiscount); err != nil {
				return nil, err
			}
		}
		in.Lines = append(in.Lines, app.LineRequest{ProductID: l.ProductId, Quantity: qty, Discount: discount})
	}
	for _, p := range req.Body.Payments {
		amount, err := parseDecimal(p.Amount, domain.ErrInvalidPaymentAmount)
		if err != nil {
			return nil, err
		}
		in.Payments = append(in.Payments, app.PaymentRequest{Method: string(p.Method), Amount: amount})
	}
	sale, err := h.uc.ConfirmSale(ctx, h.actor(ctx), in)
	if err != nil {
		return nil, err
	}
	return salesapi.ConfirmSale201JSONResponse(saleBody(sale)), nil
}

// VoidSale voids a sale and gives its stock back.
func (h *Handler) VoidSale(ctx context.Context, req salesapi.VoidSaleRequestObject) (salesapi.VoidSaleResponseObject, error) {
	sale, err := h.uc.VoidSale(ctx, h.actor(ctx), req.SaleId, req.Body.Reason)
	if err != nil {
		return nil, err
	}
	return salesapi.VoidSale200JSONResponse(saleBody(sale)), nil
}

// GetSale returns a sale with its lines, payments and totals.
func (h *Handler) GetSale(ctx context.Context, req salesapi.GetSaleRequestObject) (salesapi.GetSaleResponseObject, error) {
	sale, err := h.uc.GetSale(ctx, h.actor(ctx), req.SaleId)
	if err != nil {
		return nil, err
	}
	return salesapi.GetSale200JSONResponse(saleBody(sale)), nil
}

// ListSales returns the sales of a day in Colombia.
func (h *Handler) ListSales(ctx context.Context, req salesapi.ListSalesRequestObject) (salesapi.ListSalesResponseObject, error) {
	in := app.ListSalesInput{LocationID: req.Params.LocationId, Limit: defaultPageSize}
	if req.Params.Date != nil {
		in.Date = req.Params.Date.Time
	}
	if req.Params.Limit != nil {
		in.Limit = *req.Params.Limit
	}
	if req.Params.Offset != nil {
		in.Offset = *req.Params.Offset
	}
	page, err := h.uc.ListSales(ctx, h.actor(ctx), in)
	if err != nil {
		return nil, err
	}
	items := make([]salesapi.SaleSummary, 0, len(page.Items))
	for _, s := range page.Items {
		items = append(items, salesapi.SaleSummary{
			Id: s.ID, Number: s.Number, LocationId: s.LocationID, UserId: s.UserID,
			Status: salesapi.SaleStatus(s.Status), Total: formatMoney(s.Total),
			ConfirmedAt: s.ConfirmedAt, VoidedAt: s.VoidedAt,
		})
	}
	return salesapi.ListSales200JSONResponse{Items: items, Total: page.Total}, nil
}

func saleBody(s *domain.Sale) salesapi.Sale {
	totals := s.Totals()
	body := salesapi.Sale{
		Id: s.ID(), Number: s.Number(), LocationId: s.LocationID(), UserId: s.UserID(),
		Status: salesapi.SaleStatus(s.Status()),
		Lines:  make([]salesapi.SaleLine, 0, len(s.Lines())), Payments: make([]salesapi.SalePayment, 0, len(s.Payments())),
		Totals: salesapi.SaleTotals{
			Subtotal: formatMoney(totals.Subtotal), Discounts: formatMoney(totals.Discounts),
			Tax: formatMoney(totals.Tax), Total: formatMoney(totals.Total),
			Taxes: make([]salesapi.TaxBreakdown, 0, len(totals.Taxes)),
		},
		Paid: formatMoney(s.Paid()), Change: formatMoney(s.Change()),
	}
	if at := s.ConfirmedAt(); at != nil {
		body.ConfirmedAt = *at
	}
	for _, l := range s.Lines() {
		body.Lines = append(body.Lines, salesapi.SaleLine{
			ProductId: l.ProductID(), ProductName: l.ProductName(), Quantity: formatDecimal(l.Quantity()),
			UnitPrice: formatMoney(l.UnitPrice()), TaxRate: formatDecimal(l.TaxRate()), Discount: formatMoney(l.Discount()),
			Gross: formatMoney(l.Gross()), Base: formatMoney(l.Base()), Tax: formatMoney(l.Tax()), Total: formatMoney(l.Total()),
		})
	}
	for _, p := range s.Payments() {
		body.Payments = append(body.Payments, salesapi.SalePayment{Method: salesapi.PaymentMethod(p.Method()), Amount: formatMoney(p.Amount())})
	}
	for _, t := range totals.Taxes {
		body.Totals.Taxes = append(body.Totals.Taxes, salesapi.TaxBreakdown{
			Rate: formatDecimal(t.Rate), Base: formatMoney(t.Base), Tax: formatMoney(t.Tax),
		})
	}
	if v := s.VoidInfo(); v != nil {
		body.Void = &salesapi.SaleVoid{UserId: v.UserID, Reason: v.Reason, VoidedAt: v.At}
	}
	return body
}
