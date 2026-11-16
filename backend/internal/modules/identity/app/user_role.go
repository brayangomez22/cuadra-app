package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// UserRole returns the current role of userID in tenantID, or
// domain.ErrUserNotFound. It is for the audit log of the operations a
// background job runs on behalf of a user (an import), which have no
// access token to read the role from.
func (s *Service) UserRole(ctx context.Context, tenantID, userID uuid.UUID) (_ domain.Role, err error) {
	ctx, span := s.startSpan(ctx, "UserRole")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()), attribute.String("user.id", userID.String()))

	u, err := s.Users.GetByID(ctx, tenantID, userID)
	if err != nil {
		return "", err
	}
	return u.Role(), nil
}
