package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// Page size bounds of ListImportErrors.
const (
	defaultErrorPage = 50
	maxErrorPage     = 100
)

// ErrorPage is one page of an import's row errors and how many it has.
type ErrorPage struct {
	Items []domain.RowError
	Total int
}

// GetImport returns an import of the actor's tenant.
func (s *Service) GetImport(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Import, err error) {
	ctx, span := s.startSpan(ctx, "GetImport", a, importAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.Imports.Get(ctx, a.TenantID, id)
}

// ListImportErrors returns one page of an import's row errors, by row. The
// page size defaults to 50 and is capped at 100.
func (s *Service) ListImportErrors(ctx context.Context, a Actor, id uuid.UUID, limit, offset int) (_ ErrorPage, err error) {
	ctx, span := s.startSpan(ctx, "ListImportErrors", a, importAttr(id))
	defer func() { usecase.End(span, err) }()

	if limit <= 0 {
		limit = defaultErrorPage
	}
	items, total, err := s.Imports.Errors(ctx, a.TenantID, id, min(limit, maxErrorPage), max(offset, 0))
	if err != nil {
		return ErrorPage{}, err
	}
	return ErrorPage{Items: items, Total: total}, nil
}
