package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// FileStore implements domain.FileStore in the database, under the
// tenant's Row-Level Security: the documents' files are small (a few
// hundred KiB) and must outlive any container. Another store (a blob
// storage) can replace it behind the same port if a measurement asks.
type FileStore struct{ db *db.DB }

// NewFileStore returns a FileStore on d.
func NewFileStore(d *db.DB) *FileStore { return &FileStore{db: d} }

// Put stores a document's file, replacing the one of its kind.
func (s *FileStore) Put(ctx context.Context, tenantID, documentID uuid.UUID, f domain.File) error {
	err := s.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).PutDocumentFile(ctx, sqlcgen.PutDocumentFileParams{
			TenantID: tenantID, DocumentID: documentID, Kind: string(f.Kind), Content: f.Content,
		})
	})
	if err != nil {
		return fmt.Errorf("invoicing: put document file: %w", err)
	}
	return nil
}

// Get returns a document's file, or domain.ErrFileNotFound.
func (s *FileStore) Get(ctx context.Context, tenantID, documentID uuid.UUID, kind domain.FileKind) (domain.File, error) {
	content, err := db.Read(ctx, s.db, tenantID, func(tx pgx.Tx) ([]byte, error) {
		return sqlcgen.New(tx).GetDocumentFile(ctx, sqlcgen.GetDocumentFileParams{
			TenantID: tenantID, DocumentID: documentID, Kind: string(kind),
		})
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.File{}, domain.ErrFileNotFound
	case err != nil:
		return domain.File{}, fmt.Errorf("invoicing: get document file: %w", err)
	}
	return domain.File{Kind: kind, Content: content}, nil
}
