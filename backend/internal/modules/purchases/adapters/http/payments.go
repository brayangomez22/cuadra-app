package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/http/purchasesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// PaymentUseCases are the use cases of the payments to suppliers.
type PaymentUseCases interface {
	RecordSupplierPayment(ctx context.Context, a app.Actor, purchaseID uuid.UUID, in app.SupplierPaymentRequest) (app.SupplierPaymentResult, error)
	PurchasePayments(ctx context.Context, a app.Actor, purchaseID uuid.UUID) ([]*domain.SupplierPayment, error)
	ListPayables(ctx context.Context, a app.Actor, q app.PayablesQuery) (domain.PayablesPage, error)
}

// paymentHandlers are the operations on what the tenant owes its suppliers.
type paymentHandlers struct{ uc PaymentUseCases }

// RecordSupplierPayment records a payment to the supplier of a purchase.
func (h *paymentHandlers) RecordSupplierPayment(ctx context.Context, req purchasesapi.RecordSupplierPaymentRequestObject) (purchasesapi.RecordSupplierPaymentResponseObject, error) {
	amount, err := httpx.ParseDecimal(req.Body.Amount, domain.ErrInvalidPaymentAmount)
	if err != nil {
		return nil, err
	}
	res, err := h.uc.RecordSupplierPayment(ctx, actor(ctx), req.PurchaseId, app.SupplierPaymentRequest{
		Method: string(req.Body.Method), Amount: amount, PaidOn: dateOf(req.Body.PaidOn),
	})
	if err != nil {
		return nil, err
	}
	return purchasesapi.RecordSupplierPayment201JSONResponse(paymentBody(res.Payment)), nil
}

// ListSupplierPayments returns the payments of a purchase.
func (h *paymentHandlers) ListSupplierPayments(ctx context.Context, req purchasesapi.ListSupplierPaymentsRequestObject) (purchasesapi.ListSupplierPaymentsResponseObject, error) {
	payments, err := h.uc.PurchasePayments(ctx, actor(ctx), req.PurchaseId)
	if err != nil {
		return nil, err
	}
	items := make([]purchasesapi.SupplierPayment, 0, len(payments))
	for _, p := range payments {
		items = append(items, paymentBody(p))
	}
	return purchasesapi.ListSupplierPayments200JSONResponse{Items: items}, nil
}

// ListPayables returns what the tenant owes each supplier.
func (h *paymentHandlers) ListPayables(ctx context.Context, req purchasesapi.ListPayablesRequestObject) (purchasesapi.ListPayablesResponseObject, error) {
	q := app.PayablesQuery{Limit: defaultPageSize}
	if req.Params.Limit != nil {
		q.Limit = *req.Params.Limit
	}
	if req.Params.Offset != nil {
		q.Offset = *req.Params.Offset
	}
	page, err := h.uc.ListPayables(ctx, actor(ctx), q)
	if err != nil {
		return nil, err
	}
	t := page.Totals
	body := purchasesapi.ListPayables200JSONResponse{
		Items: make([]purchasesapi.SupplierPayable, 0, len(page.Items)), Total: t.Suppliers,
		Totals: purchasesapi.PayablesTotals{
			Balance: httpx.FormatMoney(t.Balance), Overdue: httpx.FormatMoney(t.Overdue),
			DueSoon: httpx.FormatMoney(t.DueSoon), Suppliers: t.Suppliers,
		},
	}
	for _, p := range page.Items {
		body.Items = append(body.Items, purchasesapi.SupplierPayable{
			SupplierId: p.SupplierID, SupplierName: p.SupplierName, Balance: httpx.FormatMoney(p.Balance),
			Overdue: httpx.FormatMoney(p.Overdue), DueSoon: httpx.FormatMoney(p.DueSoon), Purchases: p.Purchases,
			NextDue: date(p.NextDue),
		})
	}
	return body, nil
}

func paymentBody(p *domain.SupplierPayment) purchasesapi.SupplierPayment {
	return purchasesapi.SupplierPayment{
		Id: p.ID(), PurchaseId: p.PurchaseID(), SupplierId: p.SupplierID(),
		Method: purchasesapi.SupplierPaymentMethod(p.Method()), Amount: httpx.FormatMoney(p.Amount()),
		BalanceAfter: httpx.FormatMoney(p.BalanceAfter()), PaidOn: date(p.PaidOn()), CreatedBy: p.UserID(),
		CreatedAt: p.CreatedAt(),
	}
}
