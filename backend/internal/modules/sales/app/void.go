package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// VoidSale voids a confirmed sale and gives its stock back, in one
// transaction: if the stock cannot be given back, the sale stays confirmed.
// The actor is who voids it; reason is required. A credit sale also cancels
// its receivable, which is refused once the customer paid part of it.
func (s *Service) VoidSale(ctx context.Context, a Actor, id uuid.UUID, reason string) (_ *domain.Sale, err error) {
	ctx, span := s.startSpan(ctx, "VoidSale", a, saleAttr(id))
	defer func() { usecase.End(span, err) }()

	sale, err := s.Sales.Void(ctx, a.TenantID, id, func(ctx context.Context, sale *domain.Sale) error {
		if err := sale.Void(a.UserID, reason, s.Now()); err != nil {
			return err
		}
		if err := s.Stock.ReverseSale(ctx, a.TenantID, a.UserID, sale.ID()); err != nil {
			return err
		}
		if sale.CreditAmount().IsZero() {
			return nil
		}
		return s.Credit.CancelSale(ctx, a.TenantID, a.UserID, sale.ID())
	})
	if err != nil {
		return nil, err
	}
	s.voided.Add(ctx, 1)
	return sale, nil
}
