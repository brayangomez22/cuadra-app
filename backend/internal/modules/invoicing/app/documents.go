package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ListDocuments returns a page of the tenant's documents, the newest first.
func (s *Service) ListDocuments(ctx context.Context, a Actor, f domain.DocumentFilter) (_ domain.DocumentPage, err error) {
	ctx, span := s.startSpan(ctx, "ListDocuments", a)
	defer func() { usecase.End(span, err) }()

	return s.Documents.List(ctx, a.TenantID, f)
}

// GetDocument returns a document of the tenant.
func (s *Service) GetDocument(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Document, err error) {
	ctx, span := s.startSpan(ctx, "GetDocument", a, documentAttr(id))
	defer func() { usecase.End(span, err) }()

	return s.Documents.Get(ctx, a.TenantID, id)
}
