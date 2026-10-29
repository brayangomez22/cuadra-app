package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/http/receivablesapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// CreditUseCases are the use cases of a customer's credit.
type CreditUseCases interface {
	CustomerCredit(ctx context.Context, a app.Actor, customerID uuid.UUID) (app.CreditView, error)
	SetCreditTerms(ctx context.Context, a app.Actor, customerID uuid.UUID, in app.TermsRequest) (*domain.Account, error)
	Statement(ctx context.Context, a app.Actor, customerID uuid.UUID, limit, offset int) (app.StatementPage, error)
}

// creditHandlers are the operations on a customer's credit.
type creditHandlers struct{ uc CreditUseCases }

// GetCustomerCredit returns a customer's credit.
func (h *creditHandlers) GetCustomerCredit(ctx context.Context, req receivablesapi.GetCustomerCreditRequestObject) (receivablesapi.GetCustomerCreditResponseObject, error) {
	view, err := h.uc.CustomerCredit(ctx, actor(ctx), req.CustomerId)
	if err != nil {
		return nil, err
	}
	return receivablesapi.GetCustomerCredit200JSONResponse(creditBody(view.Account, view.Open)), nil
}

// SetCustomerCredit sets a customer's limit and term.
func (h *creditHandlers) SetCustomerCredit(ctx context.Context, req receivablesapi.SetCustomerCreditRequestObject) (receivablesapi.SetCustomerCreditResponseObject, error) {
	limit, err := httpx.ParseDecimal(req.Body.CreditLimit, domain.ErrInvalidCreditLimit)
	if err != nil {
		return nil, err
	}
	a := actor(ctx)
	if _, err := h.uc.SetCreditTerms(ctx, a, req.CustomerId, app.TermsRequest{Limit: limit, Days: req.Body.TermDays}); err != nil {
		return nil, err
	}
	view, err := h.uc.CustomerCredit(ctx, a, req.CustomerId)
	if err != nil {
		return nil, err
	}
	return receivablesapi.SetCustomerCredit200JSONResponse(creditBody(view.Account, view.Open)), nil
}

// ListCustomerCreditMovements returns a page of a customer's statement.
func (h *creditHandlers) ListCustomerCreditMovements(ctx context.Context, req receivablesapi.ListCustomerCreditMovementsRequestObject) (receivablesapi.ListCustomerCreditMovementsResponseObject, error) {
	limit, offset := page(req.Params.Limit, req.Params.Offset)
	p, err := h.uc.Statement(ctx, actor(ctx), req.CustomerId, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]receivablesapi.CreditMovement, 0, len(p.Movements))
	for _, m := range p.Movements {
		items = append(items, movementBody(m))
	}
	return receivablesapi.ListCustomerCreditMovements200JSONResponse{
		Items: items, Total: p.Total, Balance: httpx.FormatMoney(p.Account.Balance()),
	}, nil
}
