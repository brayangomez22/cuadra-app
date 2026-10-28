package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// CreateCustomer stores a new customer. Its document must be unique in the
// tenant and its segment, if any, the tenant's.
func (s *Service) CreateCustomer(ctx context.Context, a Actor, p domain.CustomerParams) (_ *domain.Customer, err error) {
	ctx, span := s.startSpan(ctx, "CreateCustomer", a)
	defer func() { usecase.End(span, err) }()

	c, err := domain.NewCustomer(a.TenantID, p, s.Now())
	if err != nil {
		return nil, err
	}
	span.SetAttributes(customerAttr(c.ID()))
	if err := s.Customers.Create(ctx, c); err != nil {
		return nil, err
	}
	s.created.Add(ctx, 1, metric.WithAttributes(attribute.String("document_type", string(c.Document().Type()))))
	return c, nil
}

// UpdateCustomer replaces a customer's data, including whether it is
// active. Its sales keep naming it.
func (s *Service) UpdateCustomer(ctx context.Context, a Actor, id uuid.UUID, p domain.CustomerParams) (_ *domain.Customer, err error) {
	ctx, span := s.startSpan(ctx, "UpdateCustomer", a, customerAttr(id))
	defer func() { usecase.End(span, err) }()

	c, err := s.Customers.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := c.Update(p); err != nil {
		return nil, err
	}
	if err := s.Customers.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// GetCustomer returns a customer of the tenant.
func (s *Service) GetCustomer(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Customer, err error) {
	ctx, span := s.startSpan(ctx, "GetCustomer", a, customerAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.Customers.GetByID(ctx, a.TenantID, id)
}

// CheckCustomer tells whether a sale can be made to customer id: it is the
// tenant's (ErrCustomerNotFound) and active (ErrCustomerInactive).
func (s *Service) CheckCustomer(ctx context.Context, a Actor, id uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "CheckCustomer", a, customerAttr(id))
	defer func() { usecase.End(span, err) }()

	c, err := s.Customers.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return err
	}
	if !c.Active() {
		return domain.ErrCustomerInactive
	}
	return nil
}
