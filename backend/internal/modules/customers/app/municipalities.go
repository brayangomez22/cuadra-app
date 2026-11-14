package app

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ListMunicipalities returns the DANE's municipalities, for the customer's
// address. The catalog is the same for every tenant.
func (s *Service) ListMunicipalities(ctx context.Context, a Actor) (_ []domain.Municipality, err error) {
	ctx, span := s.startSpan(ctx, "ListMunicipalities", a)
	defer func() { usecase.End(span, err) }()

	return s.Municipalities.List(ctx)
}
