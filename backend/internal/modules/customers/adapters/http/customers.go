package httpadapter

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/http/customersapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

// defaultPageSize is the contract's default limit.
const defaultPageSize = 20

// CustomerUseCases are the use cases of the customers.
type CustomerUseCases interface {
	CreateCustomer(ctx context.Context, a app.Actor, p domain.CustomerParams) (*domain.Customer, error)
	UpdateCustomer(ctx context.Context, a app.Actor, id uuid.UUID, p domain.CustomerParams) (*domain.Customer, error)
	GetCustomer(ctx context.Context, a app.Actor, id uuid.UUID) (*domain.Customer, error)
	ListCustomers(ctx context.Context, a app.Actor, q app.CustomerQuery) (app.CustomerPage, error)
}

// customerHandlers are the operations on customers.
type customerHandlers struct{ uc CustomerUseCases }

// ListCustomers searches the tenant's customers.
func (h *customerHandlers) ListCustomers(ctx context.Context, req customersapi.ListCustomersRequestObject) (customersapi.ListCustomersResponseObject, error) {
	q := app.CustomerQuery{SegmentID: req.Params.SegmentId, Status: domain.StatusActive, Limit: defaultPageSize}
	if req.Params.Q != nil {
		q.Text = *req.Params.Q
	}
	if req.Params.Status != nil {
		q.Status = domain.CustomerStatus(*req.Params.Status)
	}
	if req.Params.Limit != nil {
		q.Limit = *req.Params.Limit
	}
	if req.Params.Offset != nil {
		q.Offset = *req.Params.Offset
	}
	page, err := h.uc.ListCustomers(ctx, actor(ctx), q)
	if err != nil {
		return nil, err
	}
	items := make([]customersapi.Customer, 0, len(page.Items))
	for _, c := range page.Items {
		items = append(items, customerBody(c))
	}
	return customersapi.ListCustomers200JSONResponse{Items: items, Total: page.Total}, nil
}

// CreateCustomer adds a customer.
func (h *customerHandlers) CreateCustomer(ctx context.Context, req customersapi.CreateCustomerRequestObject) (customersapi.CreateCustomerResponseObject, error) {
	c, err := h.uc.CreateCustomer(ctx, actor(ctx), customerParams(*req.Body))
	if err != nil {
		return nil, err
	}
	return customersapi.CreateCustomer201JSONResponse(customerBody(c)), nil
}

// GetCustomer returns a customer.
func (h *customerHandlers) GetCustomer(ctx context.Context, req customersapi.GetCustomerRequestObject) (customersapi.GetCustomerResponseObject, error) {
	c, err := h.uc.GetCustomer(ctx, actor(ctx), req.CustomerId)
	if err != nil {
		return nil, err
	}
	return customersapi.GetCustomer200JSONResponse(customerBody(c)), nil
}

// UpdateCustomer replaces a customer's data.
func (h *customerHandlers) UpdateCustomer(ctx context.Context, req customersapi.UpdateCustomerRequestObject) (customersapi.UpdateCustomerResponseObject, error) {
	c, err := h.uc.UpdateCustomer(ctx, actor(ctx), req.CustomerId, customerParams(*req.Body))
	if err != nil {
		return nil, err
	}
	return customersapi.UpdateCustomer200JSONResponse(customerBody(c)), nil
}

// customerParams translates the body; active defaults to true, as in the
// contract.
func customerParams(in customersapi.CustomerInput) domain.CustomerParams {
	p := domain.CustomerParams{
		PersonType: domain.PersonType(in.PersonType), DocumentType: domain.DocumentType(in.DocumentType),
		DocumentNumber: in.DocumentNumber, Name: in.Name, TradeName: value(in.TradeName),
		Contact: domain.Contact{
			Email: value(in.Email), Phone: value(in.Phone), Address: value(in.Address), City: value(in.City),
		},
		SegmentID: in.SegmentId, Active: in.Active == nil || *in.Active,
	}
	if in.TaxResponsibilities != nil {
		for _, r := range *in.TaxResponsibilities {
			p.TaxResponsibilities = append(p.TaxResponsibilities, string(r))
		}
	}
	return p
}

func customerBody(c *domain.Customer) customersapi.Customer {
	contact := c.Contact()
	body := customersapi.Customer{
		Id: c.ID(), PersonType: customersapi.PersonType(c.PersonType()),
		DocumentType: customersapi.DocumentType(c.Document().Type()), DocumentNumber: c.Document().Number(),
		Name: c.Name(), TradeName: optional(c.TradeName()), Email: optional(contact.Email), Phone: optional(contact.Phone),
		Address: optional(contact.Address), City: optional(contact.City), SegmentId: c.SegmentID(),
		Active: c.Active(), CreatedAt: c.CreatedAt(),
	}
	for _, r := range c.TaxResponsibilities() {
		body.TaxResponsibilities = append(body.TaxResponsibilities, customersapi.TaxResponsibility(r))
	}
	return body
}

// value returns *s, or "" when absent.
func value(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// optional returns nil for "", so empty fields are left out of the body.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
