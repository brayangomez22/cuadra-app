package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// sentTenantsQuery calls the SECURITY DEFINER function of migration 00031.
// It is written by hand: sqlc cannot type the columns of a function
// returning TABLE.
const sentTenantsQuery = `SELECT tenant_id FROM tenants_with_sent_documents($1)`

// List returns a page of the tenant's documents, the newest first.
func (r *DocumentRepository) List(ctx context.Context, tenantID uuid.UUID, f domain.DocumentFilter) (domain.DocumentPage, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.DocumentPage{}, err
	}
	var status *string
	if f.Status != nil {
		s := string(*f.Status)
		status = &s
	}
	type page struct {
		rows  []sqlcgen.ElectronicDocument
		total int64
	}
	p, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (page, error) {
		q := sqlcgen.New(tx)
		rows, err := q.ListDocuments(ctx, sqlcgen.ListDocumentsParams{
			TenantID: tenantID, Status: status, SaleID: f.SaleID, PageLimit: limit, PageOffset: offset,
		})
		if err != nil {
			return page{}, err
		}
		total, err := q.CountDocuments(ctx, sqlcgen.CountDocumentsParams{TenantID: tenantID, Status: status, SaleID: f.SaleID})
		return page{rows: rows, total: total}, err
	})
	if err != nil {
		return domain.DocumentPage{}, fmt.Errorf("invoicing: list documents: %w", err)
	}
	return domain.DocumentPage{Documents: documentsOf(p.rows), Total: int(p.total)}, nil
}

// SentTenants returns, across tenants, those with sent documents. It reads
// ids only, through invoicing_lookup.
func (r *DocumentRepository) SentTenants(ctx context.Context, limit int) ([]uuid.UUID, error) {
	var tenants []uuid.UUID
	// Not a tenant transaction: the tenants are what it looks for.
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, sentTenantsQuery, limit)
		if err != nil {
			return err
		}
		tenants, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("invoicing: tenants with sent documents: %w", err)
	}
	return tenants, nil
}

// Sent returns the tenant's sent documents, the oldest first.
func (r *DocumentRepository) Sent(ctx context.Context, tenantID uuid.UUID, limit int) ([]*domain.Document, error) {
	pageLimit, _, err := db.PageBounds(limit, 0)
	if err != nil {
		return nil, err
	}
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.ElectronicDocument, error) {
		return sqlcgen.New(tx).SentDocuments(ctx, sqlcgen.SentDocumentsParams{TenantID: tenantID, Limit: pageLimit})
	})
	if err != nil {
		return nil, fmt.Errorf("invoicing: sent documents: %w", err)
	}
	return documentsOf(rows), nil
}
