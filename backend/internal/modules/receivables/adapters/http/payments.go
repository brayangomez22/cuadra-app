package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/http/receivablesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// PaymentUseCases are the use cases of the customers' payments.
type PaymentUseCases interface {
	RecordPayment(ctx context.Context, a app.Actor, customerID uuid.UUID, in app.PaymentRequest) (app.PaymentResult, error)
}

// paymentHandlers are the operations on the customers' payments.
type paymentHandlers struct{ uc PaymentUseCases }

// RecordCustomerPayment receives a payment from a customer.
func (h *paymentHandlers) RecordCustomerPayment(ctx context.Context, req receivablesapi.RecordCustomerPaymentRequestObject) (receivablesapi.RecordCustomerPaymentResponseObject, error) {
	amount, err := httpx.ParseDecimal(req.Body.Amount, domain.ErrInvalidPaymentAmount)
	if err != nil {
		return nil, err
	}
	res, err := h.uc.RecordPayment(ctx, actor(ctx), req.CustomerId, app.PaymentRequest{Method: string(req.Body.Method), Amount: amount})
	if err != nil {
		return nil, err
	}
	return receivablesapi.RecordCustomerPayment201JSONResponse(paymentBody(res)), nil
}
