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

	if a.UnlimitedDiscount {
		return domain.NoDiscountLimit, nil
	}
	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil {
		return domain.DiscountLimit{}, err
	}
	return discountLimit(a, policy), nil
}

// discountLimit is the actor's limit: none with sales:discount, the tenant's
// otherwise.
func discountLimit(a Actor, policy domain.SalesPolicy) domain.DiscountLimit {
	if a.UnlimitedDiscount {
		return domain.NoDiscountLimit
	}
	return policy.MaxDiscount
}
