package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ConfirmImport queues a validated import: the job that writes it is
// enqueued in the same transaction that stores the confirmation.
func (s *Service) ConfirmImport(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Import, err error) {
	ctx, span := s.startSpan(ctx, "ConfirmImport", a, importAttr(id))
	defer func() { usecase.End(span, err) }()

	imp, err := s.Imports.Get(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := imp.Confirm(); err != nil {
		return nil, err
	}
	if err := s.Imports.Queue(ctx, imp); err != nil {
		return nil, err
	}
	return imp, nil
}
