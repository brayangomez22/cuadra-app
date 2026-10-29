package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// purgeableFilesQuery calls the SECURITY DEFINER function of migration
// 00028. It is written by hand: sqlc cannot type the columns of a function
// returning TABLE.
const purgeableFilesQuery = `SELECT tenant_id, import_id FROM purgeable_import_files($1, $2, $3)`

// PurgeableFiles returns, across tenants, the imports with their file that
// ended before before. It reads ids only, through import_lookup.
func (r *ImportRepository) PurgeableFiles(ctx context.Context, before time.Time, limit int) ([]domain.FileRef, error) {
	statuses := make([]string, 0, len(domain.FinishedStatuses()))
	for _, s := range domain.FinishedStatuses() {
		statuses = append(statuses, string(s))
	}
	var refs []domain.FileRef
	// Not a tenant transaction: the tenants are what it looks for.
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, purgeableFilesQuery, before, statuses, limit)
		if err != nil {
			return err
		}
		refs, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.FileRef, error) {
			var ref domain.FileRef
			err := row.Scan(&ref.TenantID, &ref.ImportID)
			return ref, err
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("imports: find purgeable files: %w", err)
	}
	return refs, nil
}

// DropFile deletes the import's file; the import and its errors stay.
// domain.ErrImportNotFound if the tenant has no such import (or it has no
// file left).
func (r *ImportRepository) DropFile(ctx context.Context, tenantID, id uuid.UUID) error {
	affected, err := db.Affected(ctx, r.db, tenantID, func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).DropImportFile(ctx, sqlcgen.DropImportFileParams{TenantID: tenantID, ID: id})
	})
	switch {
	case err != nil:
		return fmt.Errorf("imports: drop file: %w", err)
	case affected == 0:
		return domain.ErrImportNotFound
	}
	return nil
}
