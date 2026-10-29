package app

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// SupplierView is a supplier and what the tenant owes it.
type SupplierView struct {
	*domain.Supplier
	BalanceDue decimal.Decimal
}

// SupplierQuery is a page of a supplier search. Status defaults to
// domain.SuppliersActive; the HTTP adapter bounds Limit and Offset.
type SupplierQuery struct {
	Text   string
	Status domain.SupplierStatus
	Limit  int
	Offset int
}

// SupplierPage is a page of suppliers and how many match the query.
type SupplierPage struct {
	Items []SupplierView
	Total int
}

// CreateSupplier stores a new supplier. Its NIT must be unique in the
// tenant.
func (s *Service) CreateSupplier(ctx context.Context, a Actor, p domain.SupplierParams) (_ SupplierView, err error) {
	ctx, span := s.startSpan(ctx, "CreateSupplier", a)
	defer func() { usecase.End(span, err) }()

	sup, err := domain.NewSupplier(a.TenantID, p, s.Now())
	if err != nil {
		return SupplierView{}, err
	}
	span.SetAttributes(supplierAttr(sup.ID()))
	if err := s.Suppliers.Create(ctx, sup); err != nil {
		return SupplierView{}, err
	}
	s.suppliersCreated.Add(ctx, 1)
	return SupplierView{Supplier: sup, BalanceDue: decimal.Zero}, nil
}

// UpdateSupplier replaces a supplier's data, including whether it is
// active. Its purchases keep naming it.
func (s *Service) UpdateSupplier(ctx context.Context, a Actor, id uuid.UUID, p domain.SupplierParams) (_ SupplierView, err error) {
	ctx, span := s.startSpan(ctx, "UpdateSupplier", a, supplierAttr(id))
	defer func() { usecase.End(span, err) }()

	sup, err := s.Suppliers.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return SupplierView{}, err
	}
	if err := sup.Update(p); err != nil {
		return SupplierView{}, err
	}
	if err := s.Suppliers.Update(ctx, sup); err != nil {
		return SupplierView{}, err
	}
	views, err := s.withBalances(ctx, a, []*domain.Supplier{sup})
	if err != nil {
		return SupplierView{}, err
	}
	return views[0], nil
}

// GetSupplier returns a supplier of the tenant and what it is owed.
func (s *Service) GetSupplier(ctx context.Context, a Actor, id uuid.UUID) (_ SupplierView, err error) {
	ctx, span := s.startSpan(ctx, "GetSupplier", a, supplierAttr(id))
	defer func() { usecase.End(span, err) }()

	sup, err := s.Suppliers.GetByID(ctx, a.TenantID, id)
	if err != nil {
		return SupplierView{}, err
	}
	views, err := s.withBalances(ctx, a, []*domain.Supplier{sup})
	if err != nil {
		return SupplierView{}, err
	}
	return views[0], nil
}

// ListSuppliers searches the tenant's suppliers by name or NIT, each with
// what it is owed.
func (s *Service) ListSuppliers(ctx context.Context, a Actor, q SupplierQuery) (_ SupplierPage, err error) {
	ctx, span := s.startSpan(ctx, "ListSuppliers", a)
	defer func() { usecase.End(span, err) }()

	status := q.Status
	switch status {
	case "":
		status = domain.SuppliersActive
	case domain.SuppliersActive, domain.SuppliersInactive, domain.SuppliersAll:
	default:
		return SupplierPage{}, ErrInvalidStatus
	}
	items, total, err := s.Suppliers.Search(ctx, domain.SupplierFilter{
		TenantID: a.TenantID, Text: strings.TrimSpace(q.Text), Status: status, Limit: q.Limit, Offset: q.Offset,
	})
	if err != nil {
		return SupplierPage{}, err
	}
	views, err := s.withBalances(ctx, a, items)
	if err != nil {
		return SupplierPage{}, err
	}
	return SupplierPage{Items: views, Total: total}, nil
}

// withBalances adds to each supplier what the tenant owes it.
func (s *Service) withBalances(ctx context.Context, a Actor, sups []*domain.Supplier) ([]SupplierView, error) {
	ids := make([]uuid.UUID, len(sups))
	for i, sup := range sups {
		ids[i] = sup.ID()
	}
	balances, err := s.Purchases.BalancesDue(ctx, a.TenantID, ids)
	if err != nil {
		return nil, err
	}
	views := make([]SupplierView, len(sups))
	for i, sup := range sups {
		b, ok := balances[sup.ID()]
		if !ok {
			b = decimal.Zero
		}
		views[i] = SupplierView{Supplier: sup, BalanceDue: b}
	}
	return views, nil
}
