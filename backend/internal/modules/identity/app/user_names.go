package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// UserNames returns the name of each user of tenantID among ids. It is for
// other modules, which keep users by id: the sales module names the cashier
// of each cash session.
func (s *Service) UserNames(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (_ map[uuid.UUID]string, err error) {
	ctx, span := s.startSpan(ctx, "UserNames")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()), attribute.Int("users.requested", len(ids)))

	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	return s.Users.Names(ctx, tenantID, ids)
}
