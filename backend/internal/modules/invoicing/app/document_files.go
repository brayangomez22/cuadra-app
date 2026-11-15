package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// DocumentFile returns a file of a document of the actor's tenant:
// ErrInvalidFileKind, ErrDocumentNotFound or, if the provider did not
// return it (yet), ErrFileNotFound.
func (s *Service) DocumentFile(ctx context.Context, a Actor, documentID uuid.UUID, kind string) (_ domain.File, err error) {
	ctx, span := s.startSpan(ctx, "GetDocumentFile", a, documentAttr(documentID))
	defer func() { usecase.End(span, err) }()

	k, err := domain.ParseFileKind(kind)
	if err != nil {
		return domain.File{}, err
	}
	if _, err := s.Documents.Get(ctx, a.TenantID, documentID); err != nil {
		return domain.File{}, err
	}
	return s.Files.Get(ctx, a.TenantID, documentID, k)
}
