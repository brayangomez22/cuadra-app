package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// CustomerNames returns the names of the tenant's customers among ids,
// active or not, for the reports of other modules (the receivables aging).
// The ones that are not the tenant's are left out.
func (s *Service) CustomerNames(ctx context.Context, a Actor, ids []uuid.UUID) (_ map[uuid.UUID]string, err error) {
	ctx, span := s.startSpan(ctx, "CustomerNames", a)
	defer func() { usecase.End(span, err) }()

	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	return s.Customers.Names(ctx, a.TenantID, ids)
}
