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

// Enqueue enqueues the issue of a document in tx, the transaction that
// stores it (outbox); the jobs adapter gives it, so this package does not
// know River.
type Enqueue func(ctx context.Context, tx pgx.Tx, tenantID, documentID uuid.UUID) error

// DocumentRepository implements domain.DocumentRepository.
type DocumentRepository struct {
	db      *db.DB
	enqueue Enqueue
}

// NewDocumentRepository returns a DocumentRepository on d that enqueues the
// issue of the documents with enqueue.
func NewDocumentRepository(d *db.DB, enqueue Enqueue) *DocumentRepository {
	return &DocumentRepository{db: d, enqueue: enqueue}
}

// Add stores a new document and enqueues its issue in the same transaction.
func (r *DocumentRepository) Add(ctx context.Context, d *domain.Document) error {
	s := d.Snapshot()
	err := r.db.WithTenantTx(ctx, s.TenantID, func(tx pgx.Tx) error {
		if err := sqlcgen.New(tx).AddDocument(ctx, addParams(s)); err != nil {
			return err
		}
		return r.enqueue(ctx, tx, s.TenantID, s.ID)
	})
	if err != nil {
		return fmt.Errorf("invoicing: add document: %w", err)
	}
	return nil
}

// Get returns a document of the tenant.
func (r *DocumentRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Document, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.ElectronicDocument, error) {
		return sqlcgen.New(tx).GetDocument(ctx, sqlcgen.GetDocumentParams{TenantID: tenantID, ID: id})
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrDocumentNotFound
	case err != nil:
		return nil, fmt.Errorf("invoicing: get document: %w", err)
	}
	return documentOf(row), nil
}

// Save stores the document's state.
func (r *DocumentRepository) Save(ctx context.Context, d *domain.Document) error {
	return r.save(ctx, d, false)
}

// Requeue stores the document's state and enqueues its issue again, in one
// transaction.
func (r *DocumentRepository) Requeue(ctx context.Context, d *domain.Document) error {
	return r.save(ctx, d, true)
}

func (r *DocumentRepository) save(ctx context.Context, d *domain.Document, enqueue bool) error {
	s := d.Snapshot()
	err := r.db.WithTenantTx(ctx, s.TenantID, func(tx pgx.Tx) error {
		n, err := sqlcgen.New(tx).SaveDocument(ctx, saveParams(s))
		switch {
		case err != nil:
			return err
		case n == 0:
			return domain.ErrDocumentNotFound
		case enqueue:
			return r.enqueue(ctx, tx, s.TenantID, s.ID)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("invoicing: save document: %w", err)
	}
	return nil
}

// ExistsForSale is whether the sale has a document.
func (r *DocumentRepository) ExistsForSale(ctx context.Context, tenantID, saleID uuid.UUID) (bool, error) {
	found, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (bool, error) {
		return sqlcgen.New(tx).DocumentExistsForSale(ctx, sqlcgen.DocumentExistsForSaleParams{TenantID: tenantID, SaleID: saleID})
	})
	if err != nil {
		return false, fmt.Errorf("invoicing: document of sale: %w", err)
	}
	return found, nil
}
