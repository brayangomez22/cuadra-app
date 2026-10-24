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

// UseCases are the sales and cash use cases the handlers call (*app.Service).
type UseCases interface {
	ConfirmSale(ctx context.Context, a app.Actor, in app.ConfirmSaleInput) (*domain.Sale, error)
	VoidSale(ctx context.Context, a app.Actor, id uuid.UUID, reason string) (*domain.Sale, error)
	GetSale(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Sale, error)
	ListSales(ctx context.Context, a app.Actor, in app.ListSalesInput) (domain.SalePage, error)
	GetDiscountLimit(ctx context.Context, a app.Actor) (domain.DiscountLimit, error)

	OpenCashSession(ctx context.Context, a app.Actor, in app.OpenCashSessionInput) (*domain.CashSession, error)
	RecordCashMovement(ctx context.Context, a app.Actor, sessionID uuid.UUID, in app.CashMovementRequest) (domain.CashMovement, error)
	CloseCashSession(ctx context.Context, a app.Actor, sessionID uuid.UUID, in app.CloseCashSessionInput) (app.CashSessionView, error)
	GetCashSession(ctx context.Context, a app.Actor, id uuid.UUID) (app.CashSessionView, error)
	GetCurrentCashSession(ctx context.Context, a app.Actor) (app.CurrentCashSession, error)
}

// Handler implements salesapi.StrictServerInterface.
type Handler struct {
	uc  UseCases
	can auth.Authorizer
}

var _ salesapi.StrictServerInterface = (*Handler)(nil)

// Register mounts the sales and cash operations on mux, with the module's error
// mapping. can decides who discounts without limit.
func Register(mux *http.ServeMux, uc UseCases, can auth.Authorizer, log *slog.Logger) {
	h := &Handler{uc: uc, can: can}
	errs := httpx.NewErrorMapper(log, errorRules...)
	salesapi.HandlerWithOptions(
		salesapi.NewStrictHandlerWithOptions(h, []salesapi.StrictMiddlewareFunc{auth.RequirePrincipal[salesapi.StrictHandlerFunc]}, salesapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  errs.RequestError,
			ResponseErrorHandlerFunc: errs.ResponseError,
		}),
		salesapi.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: errs.RequestError},
	)
}

// actor returns who calls: tenant and user come only from the access token,
// and whether they discount without limit from their role.
func (h *Handler) actor(ctx context.Context) app.Actor {
	p, _ := auth.PrincipalFrom(ctx)
	return app.Actor{TenantID: p.TenantID, UserID: p.UserID, UnlimitedDiscount: h.can(p.Role, permissionSalesDiscount)}
}

// ConfirmSale checks out a cart.
func (h *Handler) ConfirmSale(ctx context.Context, req salesapi.ConfirmSaleRequestObject) (salesapi.ConfirmSaleResponseObject, error) {
	// The POS latency SLO measures charging a sale (docs/slo.md).
	httpx.SetOperation(ctx, "pos.confirm_sale")
	in := app.ConfirmSaleInput{LocationID: req.Body.LocationId}
	for _, l := range req.Body.Lines {
		qty, err := httpx.ParseDecimal(l.Quantity, domain.ErrInvalidQuantity)
		if err != nil {
			return nil, err
		}
		discount := decimal.Zero
		if l.Discount != nil {
			if discount, err = httpx.ParseDecimal(*l.Discount, domain.ErrInvalidDiscount); err != nil {
				return nil, err
			}
		}
		in.Lines = append(in.Lines, app.LineRequest{
			VariantID: l.VariantId, Quantity: qty, Discount: discount, UnitID: l.UnitId,
		})
	}
	for _, p := range req.Body.Payments {
		amount, err := httpx.ParseDecimal(p.Amount, domain.ErrInvalidPaymentAmount)
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
			Status: salesapi.SaleStatus(s.Status), Total: httpx.FormatMoney(s.Total),
			ConfirmedAt: s.ConfirmedAt, VoidedAt: s.VoidedAt,
		})
	}
	return salesapi.ListSales200JSONResponse{Items: items, Total: page.Total}, nil
}

// GetDiscountLimit returns the caller's discount limit, as a fraction of a
// line's gross amount; without limit, the whole line.
func (h *Handler) GetDiscountLimit(ctx context.Context, _ salesapi.GetDiscountLimitRequestObject) (salesapi.GetDiscountLimitResponseObject, error) {
	limit, err := h.uc.GetDiscountLimit(ctx, h.actor(ctx))
	if err != nil {
		return nil, err
	}
	if limit.Unlimited() {
		return salesapi.GetDiscountLimit200JSONResponse{Unlimited: true, Max: "1"}, nil
	}
	return salesapi.GetDiscountLimit200JSONResponse{Max: httpx.FormatDecimal(limit.Max())}, nil
}

func saleBody(s *domain.Sale) salesapi.Sale {
	totals := s.Totals()
	body := salesapi.Sale{
		Id: s.ID(), Number: s.Number(), LocationId: s.LocationID(), UserId: s.UserID(),
		Status: salesapi.SaleStatus(s.Status()), CashSessionId: s.CashSessionID(),
		Lines: make([]salesapi.SaleLine, 0, len(s.Lines())), Payments: make([]salesapi.SalePayment, 0, len(s.Payments())),
		Totals: salesapi.SaleTotals{
			Subtotal: httpx.FormatMoney(totals.Subtotal), Discounts: httpx.FormatMoney(totals.Discounts),
			Tax: httpx.FormatMoney(totals.Tax), Total: httpx.FormatMoney(totals.Total),
			Taxes: make([]salesapi.TaxBreakdown, 0, len(totals.Taxes)),
		},
		Paid: httpx.FormatMoney(s.Paid()), Change: httpx.FormatMoney(s.Change()),
	}
	if at := s.ConfirmedAt(); at != nil {
		body.ConfirmedAt = *at
	}
	for _, l := range s.Lines() {
		line := salesapi.SaleLine{
			VariantId: l.VariantID(), ProductId: l.ProductID(), ProductName: l.ProductName(), Quantity: httpx.FormatDecimal(l.Quantity()),
			UnitFactor: httpx.FormatDecimal(l.Unit().Factor), BaseQuantity: httpx.FormatDecimal(l.BaseQuantity()),
			UnitPrice: httpx.FormatMoney(l.UnitPrice()), PriceRule: string(l.PriceRule()), TaxRate: httpx.FormatDecimal(l.TaxRate()),
			Discount: httpx.FormatMoney(l.Discount()), Gross: httpx.FormatMoney(l.Gross()), Base: httpx.FormatMoney(l.Base()),
			Tax: httpx.FormatMoney(l.Tax()), Total: httpx.FormatMoney(l.Total()),
		}
		if label := l.VariantLabel(); label != "" {
			line.VariantLabel = &label
		}
		if u := l.Unit(); !u.IsBase() {
			line.UnitCode = &u.Code
		}
		body.Lines = append(body.Lines, line)
	}
	for _, p := range s.Payments() {
		body.Payments = append(body.Payments, salesapi.SalePayment{Method: salesapi.PaymentMethod(p.Method()), Amount: httpx.FormatMoney(p.Amount())})
	}
	for _, t := range totals.Taxes {
		body.Totals.Taxes = append(body.Totals.Taxes, salesapi.TaxBreakdown{
			Rate: httpx.FormatDecimal(t.Rate), Base: httpx.FormatMoney(t.Base), Tax: httpx.FormatMoney(t.Tax),
		})
	}
	if v := s.VoidInfo(); v != nil {
		body.Void = &salesapi.SaleVoid{UserId: v.UserID, Reason: v.Reason, VoidedAt: v.At}
	}
	return body
}
