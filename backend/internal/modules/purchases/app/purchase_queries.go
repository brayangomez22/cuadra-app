package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// PurchaseQuery is a page of the tenant's purchases. Status "" lists every
// status; the HTTP adapter bounds Limit and Offset.
type PurchaseQuery struct {
	SupplierID *uuid.UUID
	Status     domain.PurchaseStatus
	From       *domain.Date
	To         *domain.Date
	Limit      int
	Offset     int
}

// PurchasePage is a page of purchases and how many match the query.
type PurchasePage struct {
	Items []domain.PurchaseSummary
	Total int
}

// GetPurchase returns a purchase of the tenant with its lines.
func (s *Service) GetPurchase(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Purchase, err error) {
	ctx, span := s.startSpan(ctx, "GetPurchase", a, purchaseAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.Purchases.GetByID(ctx, a.TenantID, id)
}

// ListPurchases lists the tenant's purchases, the latest invoice first.
func (s *Service) ListPurchases(ctx context.Context, a Actor, q PurchaseQuery) (_ PurchasePage, err error) {
	ctx, span := s.startSpan(ctx, "ListPurchases", a)
	defer func() { usecase.End(span, err) }()

	switch q.Status {
	case "", domain.PurchaseOpen, domain.PurchasePaid:
	default:
		return PurchasePage{}, ErrInvalidStatus
	}
	items, total, err := s.Purchases.Search(ctx, domain.PurchaseFilter{
		TenantID: a.TenantID, SupplierID: q.SupplierID, Status: q.Status, From: q.From, To: q.To,
		Limit: q.Limit, Offset: q.Offset,
	})
	if err != nil {
		return PurchasePage{}, err
	}
	return PurchasePage{Items: items, Total: total}, nil
}
