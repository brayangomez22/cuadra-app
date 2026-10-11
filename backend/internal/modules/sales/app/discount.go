package app

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// GetDiscountLimit returns how much the actor may discount on a line: no
// limit with sales:discount, the tenant's otherwise. The point of sale shows
// the discount field with it; ConfirmSale applies it again.
func (s *Service) GetDiscountLimit(ctx context.Context, a Actor) (_ domain.DiscountLimit, err error) {
	ctx, span := s.startSpan(ctx, "GetDiscountLimit", a)
	defer func() { endSpan(span, err) }()

	return s.discountLimit(ctx, a)
}
